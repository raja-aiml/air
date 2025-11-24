package compose

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Service represents a Docker Compose stack managed via Docker SDK
type Service struct {
	cli         *client.Client
	project     *composetypes.Project
	projectName string
	networkIDs  map[string]string // network name -> network ID
	volumeNames []string          // list of created volumes
}

// ServiceStatus represents the status of compose services
type ServiceStatus struct {
	Services map[string]ServiceInfo `json:"services"`
}

// ServiceInfo contains info about a single service
type ServiceInfo struct {
	Name        string   `json:"name"`
	State       string   `json:"state"`      // running, exited, etc.
	Health      string   `json:"health"`     // healthy, unhealthy, starting, none
	Ports       []string `json:"ports"`      // e.g., "0.0.0.0:5432->5432/tcp"
	HealthURL   string   `json:"health_url"` // Derived URL for access
	ContainerID string   `json:"container_id"`
}

// Config holds configuration for compose operations
type Config struct {
	ComposeFilePath string            // Path to docker-compose.yml
	ProjectName     string            // Docker Compose project name
	Env             map[string]string // Environment variables
}

// New creates a new compose service manager using Docker SDK
func New(cfg Config) (*Service, error) {
	absPath, err := filepath.Abs(cfg.ComposeFilePath)
	if err != nil {
		return nil, fmt.Errorf("resolve compose file path: %w", err)
	}

	// Create Docker client with proper options
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	// Verify Docker daemon is reachable
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("docker daemon not reachable (is Docker Desktop running?): %w", err)
	}

	// Load compose file using compose-spec
	configFiles := []composetypes.ConfigFile{
		{Filename: absPath},
	}

	configDetails := composetypes.ConfigDetails{
		ConfigFiles: configFiles,
		WorkingDir:  filepath.Dir(absPath),
		Environment: cfg.Env,
	}

	project, err := loader.LoadWithContext(context.Background(), configDetails, func(options *loader.Options) {
		options.SetProjectName(cfg.ProjectName, true)
	})
	if err != nil {
		return nil, fmt.Errorf("load compose file: %w", err)
	}

	return &Service{
		cli:         cli,
		project:     project,
		projectName: cfg.ProjectName,
		networkIDs:  make(map[string]string),
		volumeNames: make([]string, 0),
	}, nil
}

// Start brings up all compose services using Docker SDK
func (s *Service) Start(ctx context.Context) error {
	var startErr error

	// Cleanup on failure - rollback any partially created resources
	cleanup := func() {
		if startErr != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = s.Stop(cleanupCtx)
		}
	}
	defer cleanup()

	// 1. Create networks
	for netName, netConfig := range s.project.Networks {
		fullName := fmt.Sprintf("%s_%s", s.projectName, netName)

		// Check if network exists (use exact name match)
		existingNetworks, err := s.cli.NetworkList(ctx, network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", fmt.Sprintf("^%s$", fullName))),
		})
		if err != nil {
			startErr = fmt.Errorf("list networks: %w", err)
			return startErr
		}

		// Double-check exact name match (Docker filter may still do substring match)
		var netID string
		for _, n := range existingNetworks {
			if n.Name == fullName {
				netID = n.ID
				break
			}
		}

		if netID != "" {
			// Network already exists
		} else {
			// Create network with project label for discovery during cleanup
			labels := make(map[string]string)
			for k, v := range netConfig.Labels {
				labels[k] = v
			}
			labels["com.docker.compose.project"] = s.projectName
			labels["com.docker.compose.network"] = netName

			opts := network.CreateOptions{
				Driver: netConfig.Driver,
				Labels: labels,
			}
			if netConfig.EnableIPv6 != nil && *netConfig.EnableIPv6 {
				enableIPv6 := true
				opts.EnableIPv6 = &enableIPv6
			}
			resp, err := s.cli.NetworkCreate(ctx, fullName, opts)
			if err != nil {
				startErr = fmt.Errorf("create network %s: %w", netName, err)
				return startErr
			}
			netID = resp.ID
		}
		s.networkIDs[netName] = netID
	}

	// 2. Create volumes
	for volName, volConfig := range s.project.Volumes {
		fullName := fmt.Sprintf("%s_%s", s.projectName, volName)

		// Check if volume exists (use exact name match)
		existingVolumes, err := s.cli.VolumeList(ctx, volume.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", fmt.Sprintf("^%s$", fullName))),
		})
		if err != nil {
			startErr = fmt.Errorf("list volumes: %w", err)
			return startErr
		}

		// Double-check exact name match
		volumeExists := false
		for _, v := range existingVolumes.Volumes {
			if v.Name == fullName {
				volumeExists = true
				break
			}
		}

		if !volumeExists {
			// Create volume with project label for discovery during cleanup
			labels := make(map[string]string)
			for k, v := range volConfig.Labels {
				labels[k] = v
			}
			labels["com.docker.compose.project"] = s.projectName
			labels["com.docker.compose.volume"] = volName

			_, err := s.cli.VolumeCreate(ctx, volume.CreateOptions{
				Name:   fullName,
				Driver: volConfig.Driver,
				Labels: labels,
			})
			if err != nil {
				startErr = fmt.Errorf("create volume %s: %w", volName, err)
				return startErr
			}
		}
		s.volumeNames = append(s.volumeNames, fullName)
	}

	// 3. Start services in dependency order
	orderedServices := s.sortServicesByDependency()
	for _, svc := range orderedServices {
		if err := s.startService(ctx, svc); err != nil {
			startErr = fmt.Errorf("start service %s: %w", svc.Name, err)
			return startErr
		}
	}

	return nil
}

// sortServicesByDependency returns services sorted so dependencies start first
func (s *Service) sortServicesByDependency() []composetypes.ServiceConfig {
	// Build dependency graph
	services := make(map[string]composetypes.ServiceConfig)
	for name, svc := range s.project.Services {
		services[name] = svc
	}

	// Track which services have been added to result
	added := make(map[string]bool)
	result := make([]composetypes.ServiceConfig, 0, len(services))

	// Helper to get dependencies for a service
	getDeps := func(svc composetypes.ServiceConfig) []string {
		deps := make([]string, 0)
		for dep := range svc.DependsOn {
			deps = append(deps, dep)
		}
		return deps
	}

	// Iteratively add services whose dependencies are all satisfied
	for len(result) < len(services) {
		progress := false
		for name, svc := range services {
			if added[name] {
				continue
			}

			// Check if all dependencies are satisfied
			allDepsSatisfied := true
			for _, dep := range getDeps(svc) {
				if !added[dep] {
					allDepsSatisfied = false
					break
				}
			}

			if allDepsSatisfied {
				result = append(result, svc)
				added[name] = true
				progress = true
			}
		}

		// If no progress was made, there might be a circular dependency
		// Add remaining services anyway to avoid infinite loop
		if !progress {
			for name, svc := range services {
				if !added[name] {
					result = append(result, svc)
					added[name] = true
				}
			}
			break
		}
	}

	return result
}

// startService starts a single service container
func (s *Service) startService(ctx context.Context, svc composetypes.ServiceConfig) error {
	// Use container_name from compose file if specified, otherwise use default naming
	containerName := svc.ContainerName
	if containerName == "" {
		containerName = fmt.Sprintf("%s-%s-1", s.projectName, svc.Name)
	}

	// Check if container already exists
	containers, err := s.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	var containerID string
	if len(containers) > 0 {
		containerID = containers[0].ID
		// Start if stopped
		if containers[0].State != "running" {
			if err := s.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
				return fmt.Errorf("start existing container: %w", err)
			}
		}
		return nil
	}

	// Pull image if needed
	if _, _, err := s.cli.ImageInspectWithRaw(ctx, svc.Image); err != nil {
		reader, err := s.cli.ImagePull(ctx, svc.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("pull image %s: %w", svc.Image, err)
		}
		defer reader.Close()
		io.Copy(io.Discard, reader) // Consume pull output
	}

	// Build container config
	containerConfig := &container.Config{
		Image:  svc.Image,
		Env:    buildEnvList(svc.Environment),
		Labels: svc.Labels,
	}

	// Add command if specified
	if len(svc.Command) > 0 {
		containerConfig.Cmd = []string(svc.Command)
	}

	// Add compose labels
	if containerConfig.Labels == nil {
		containerConfig.Labels = make(map[string]string)
	}
	containerConfig.Labels["com.docker.compose.project"] = s.projectName
	containerConfig.Labels["com.docker.compose.service"] = svc.Name

	// Build port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for _, port := range svc.Ports {
		containerPort := nat.Port(fmt.Sprintf("%d/%s", port.Target, port.Protocol))
		exposedPorts[containerPort] = struct{}{}

		if port.Published != "" {
			portBindings[containerPort] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port.Published,
				},
			}
		}
	}
	containerConfig.ExposedPorts = exposedPorts

	// Build host config
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	// Add restart policy if specified
	if svc.Restart != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(svc.Restart),
		}
	}

	// Add volume mounts
	for _, vol := range svc.Volumes {
		if vol.Type == "volume" {
			fullVolName := fmt.Sprintf("%s_%s", s.projectName, vol.Source)
			hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", fullVolName, vol.Target))
		} else if vol.Type == "bind" {
			// Resolve relative paths
			source := vol.Source
			if !filepath.IsAbs(source) {
				source = filepath.Join(s.project.WorkingDir, source)
			}
			hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", source, vol.Target))
		}
	}

	// Build network config
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings),
	}
	for netName := range svc.Networks {
		fullNetName := fmt.Sprintf("%s_%s", s.projectName, netName)
		networkConfig.EndpointsConfig[fullNetName] = &network.EndpointSettings{
			Aliases: []string{svc.Name},
		}
	}

	// Create and start container
	resp, err := s.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := s.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	return nil
}

// Stop brings down all compose services
func (s *Service) Stop(ctx context.Context) error {
	// Phase 1: Stop and remove all containers (with retry logic)
	if err := s.stopContainers(ctx); err != nil {
		// Log but continue - we still want to clean up networks/volumes
		fmt.Printf("Warning: container cleanup had errors: %v\n", err)
	}

	// Phase 2: Remove networks (discovered by label, not from memory)
	if err := s.removeNetworks(ctx); err != nil {
		fmt.Printf("Warning: network cleanup had errors: %v\n", err)
	}

	// Phase 3: Remove volumes (discovered by label, not from memory)
	if err := s.removeVolumes(ctx); err != nil {
		fmt.Printf("Warning: volume cleanup had errors: %v\n", err)
	}

	return nil
}

// stopContainers stops and removes all project containers with retry logic
func (s *Service) stopContainers(ctx context.Context) error {
	listOpts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", s.projectName)),
		),
	}

	containers, err := s.cli.ContainerList(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	var lastErr error
	timeout := 10

	for _, c := range containers {
		// Check context cancellation before processing each container
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled during container cleanup: %w", err)
		}

		containerName := "unknown"
		if len(c.Names) > 0 {
			containerName = c.Names[0]
		}

		// Step 1: Try graceful stop first
		if c.State == "running" {
			if err := s.cli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout}); err != nil {
				// If graceful stop fails, force kill
				if killErr := s.cli.ContainerKill(ctx, c.ID, "SIGKILL"); killErr != nil {
					fmt.Printf("Warning: failed to kill container %s: %v\n", containerName, killErr)
				}
			}
		}

		// Step 2: Wait briefly for container state to settle (with context awareness)
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during container cleanup: %w", ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}

		// Step 3: Remove container with retry
		removed := false
		for attempt := 0; attempt < 3; attempt++ {
			// Check context before each retry
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context cancelled during container removal: %w", err)
			}

			if err := s.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{
				Force:         true,
				RemoveVolumes: true,
			}); err != nil {
				if attempt < 2 {
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled during container removal: %w", ctx.Err())
					case <-time.After(500 * time.Millisecond):
					}
					continue
				}
				lastErr = fmt.Errorf("remove container %s: %w", containerName, err)
				fmt.Printf("Warning: failed to remove container %s after retries: %v\n", containerName, err)
			} else {
				removed = true
				break
			}
		}

		if !removed && lastErr == nil {
			lastErr = fmt.Errorf("failed to remove container %s", containerName)
		}
	}

	return lastErr
}

// removeNetworks discovers and removes all project networks by label
func (s *Service) removeNetworks(ctx context.Context) error {
	// Discover networks by label (works even for fresh Service instances)
	networks, err := s.cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", s.projectName)),
		),
	})
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}

	var lastErr error
	for _, n := range networks {
		// Check context cancellation before processing each network
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled during network cleanup: %w", err)
		}

		// Retry network removal (may need time for container detachment)
		removed := false
		for attempt := 0; attempt < 3; attempt++ {
			// Check context before each retry
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context cancelled during network removal: %w", err)
			}

			if err := s.cli.NetworkRemove(ctx, n.ID); err != nil {
				if attempt < 2 {
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled during network removal: %w", ctx.Err())
					case <-time.After(500 * time.Millisecond):
					}
					continue
				}
				lastErr = fmt.Errorf("remove network %s: %w", n.Name, err)
				fmt.Printf("Warning: failed to remove network %s: %v\n", n.Name, err)
			} else {
				removed = true
				break
			}
		}
		if removed {
			// Also remove from in-memory map if present
			for name, id := range s.networkIDs {
				if id == n.ID {
					delete(s.networkIDs, name)
					break
				}
			}
		}
	}

	return lastErr
}

// removeVolumes discovers and removes all project volumes by label
func (s *Service) removeVolumes(ctx context.Context) error {
	// Discover volumes by label (works even for fresh Service instances)
	volumes, err := s.cli.VolumeList(ctx, volume.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", s.projectName)),
		),
	})
	if err != nil {
		return fmt.Errorf("list volumes: %w", err)
	}

	var lastErr error
	for _, v := range volumes.Volumes {
		if err := s.cli.VolumeRemove(ctx, v.Name, true); err != nil {
			lastErr = fmt.Errorf("remove volume %s: %w", v.Name, err)
			fmt.Printf("Warning: failed to remove volume %s: %v\n", v.Name, err)
		}
	}

	// Clear in-memory tracking
	s.volumeNames = s.volumeNames[:0]

	return lastErr
}

// Status retrieves current status of all services
func (s *Service) Status(ctx context.Context) (*ServiceStatus, error) {
	listOpts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", s.projectName)),
		),
	}

	containers, err := s.cli.ContainerList(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	status := &ServiceStatus{
		Services: make(map[string]ServiceInfo),
	}

	for _, c := range containers {
		serviceName := c.Labels["com.docker.compose.service"]
		if serviceName == "" {
			continue
		}

		// Build port list
		ports := []string{}
		for _, port := range c.Ports {
			if port.PublicPort > 0 {
				ports = append(ports, fmt.Sprintf("%s:%d->%d/%s",
					port.IP, port.PublicPort, port.PrivatePort, port.Type))
			}
		}

		// Derive health URL
		healthURL := deriveHealthURL(serviceName, ports)

		// Get health status from container inspection
		health := "none" // Default: no healthcheck configured
		inspect, err := s.cli.ContainerInspect(ctx, c.ID)
		if err == nil && inspect.State != nil && inspect.State.Health != nil {
			health = inspect.State.Health.Status // healthy, unhealthy, starting
		}

		status.Services[serviceName] = ServiceInfo{
			Name:        serviceName,
			State:       c.State,
			Health:      health,
			Ports:       ports,
			HealthURL:   healthURL,
			ContainerID: c.ID[:12],
		}
	}

	return status, nil
}

// Logs retrieves logs from a specific service
func (s *Service) Logs(ctx context.Context, serviceName string) (string, error) {
	// Find container for service
	listOpts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", s.projectName)),
			filters.Arg("label", fmt.Sprintf("com.docker.compose.service=%s", serviceName)),
		),
	}

	containers, err := s.cli.ContainerList(ctx, listOpts)
	if err != nil {
		return "", fmt.Errorf("list containers: %w", err)
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	reader, err := s.cli.ContainerLogs(ctx, containers[0].ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	})
	if err != nil {
		return "", fmt.Errorf("get logs: %w", err)
	}
	defer reader.Close()

	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read logs: %w", err)
	}

	return string(logs), nil
}

// WaitForHealthy waits for all services to be running and healthy
func (s *Service) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled while waiting for services: %w", err)
		}

		status, err := s.Status(ctx)
		if err != nil {
			return err
		}

		allHealthy := true
		for _, svc := range status.Services {
			// Container must be running
			if svc.State != "running" {
				allHealthy = false
				break
			}

			// If container has a healthcheck, it must be healthy
			// Health values: "healthy", "unhealthy", "starting", "none"
			if svc.Health != "none" && svc.Health != "healthy" {
				allHealthy = false
				break
			}
		}

		if allHealthy && len(status.Services) == len(s.project.Services) {
			return nil
		}

		// Use context-aware sleep
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for services: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for services to be healthy")
}

// Close closes the Docker client connection
func (s *Service) Close() error {
	return s.cli.Close()
}

// GetContainerLogs retrieves logs for a container by ID
func (s *Service) GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
	}
	return s.cli.ContainerLogs(ctx, containerID, options)
}

// GetClient returns the underlying Docker client
func (s *Service) GetClient() *client.Client {
	return s.cli
}

// Helper functions

func buildEnvList(envMap composetypes.MappingWithEquals) []string {
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		if v == nil {
			// Environment variable without value
			if osVal, ok := os.LookupEnv(k); ok {
				result = append(result, fmt.Sprintf("%s=%s", k, osVal))
			}
		} else {
			result = append(result, fmt.Sprintf("%s=%s", k, *v))
		}
	}
	return result
}

func deriveHealthURL(serviceName string, ports []string) string {
	if len(ports) == 0 {
		return ""
	}

	// Parse first mapped port: "0.0.0.0:5432->5432/tcp" -> extract 5432
	firstPort := ports[0]
	parts := strings.Split(firstPort, ":")
	if len(parts) < 2 {
		return ""
	}

	hostPort := strings.Split(parts[1], "->")[0]

	// Map service names to protocols
	switch serviceName {
	case "db":
		return fmt.Sprintf("postgres://localhost:%s", hostPort)
	case "jaeger":
		return fmt.Sprintf("http://localhost:%s", hostPort) // First port is UI (16686)
	case "prometheus":
		return fmt.Sprintf("http://localhost:%s", hostPort)
	case "otel-collector":
		return "" // No user-facing URL
	default:
		return fmt.Sprintf("http://localhost:%s", hostPort)
	}
}
