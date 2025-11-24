-- Detect log level from message content and set severity fields
function detect_level(tag, timestamp, record)
    local log = record["log"] or ""
    local level = record["level"] or ""

    -- Convert to lowercase for matching
    local log_lower = string.lower(log)
    local level_lower = string.lower(level)

    -- Check if level is already set in the record
    if level_lower == "error" or level_lower == "err" then
        record["severity_text"] = "ERROR"
        record["severity_number"] = 17
        return 1, timestamp, record
    elseif level_lower == "warn" or level_lower == "warning" then
        record["severity_text"] = "WARN"
        record["severity_number"] = 13
        return 1, timestamp, record
    elseif level_lower == "fatal" or level_lower == "critical" then
        record["severity_text"] = "FATAL"
        record["severity_number"] = 21
        return 1, timestamp, record
    elseif level_lower == "info" then
        record["severity_text"] = "INFO"
        record["severity_number"] = 9
        return 1, timestamp, record
    elseif level_lower == "debug" or level_lower == "trace" then
        record["severity_text"] = "DEBUG"
        record["severity_number"] = 5
        return 1, timestamp, record
    end

    -- Try to detect from log message content
    -- Check for error patterns
    if string.find(log_lower, "error") or
       string.find(log_lower, "exception") or
       string.find(log_lower, "failed") or
       string.find(log_lower, "failure") or
       string.find(log_lower, "panic") then
        record["severity_text"] = "ERROR"
        record["severity_number"] = 17
        return 1, timestamp, record
    end

    -- Check for warning patterns
    if string.find(log_lower, "warn") or
       string.find(log_lower, "warning") or
       string.find(log_lower, "deprecated") then
        record["severity_text"] = "WARN"
        record["severity_number"] = 13
        return 1, timestamp, record
    end

    -- Check for fatal patterns
    if string.find(log_lower, "fatal") or
       string.find(log_lower, "critical") then
        record["severity_text"] = "FATAL"
        record["severity_number"] = 21
        return 1, timestamp, record
    end

    -- Default to INFO
    record["severity_text"] = "INFO"
    record["severity_number"] = 9
    return 1, timestamp, record
end
