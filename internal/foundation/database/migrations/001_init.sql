-- Enable pgvector
CREATE EXTENSION IF NOT EXISTS vector;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Question bank with embeddings
CREATE TABLE IF NOT EXISTS question_bank (
    id uuid PRIMARY KEY,
    question_json JSONB NOT NULL,
    concepts TEXT[] NOT NULL CHECK (array_length(concepts, 1) > 0),
    difficulty INT NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
    embedding VECTOR(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Concept mastery per user/concept
CREATE TABLE IF NOT EXISTS concept_mastery (
    user_id uuid NOT NULL,
    concept TEXT NOT NULL,
    mastery_score FLOAT NOT NULL,
    embedding VECTOR(1536),
    PRIMARY KEY (user_id, concept)
);

-- User progress history
CREATE TABLE IF NOT EXISTS user_progress (
    user_id uuid NOT NULL,
    question_id uuid NOT NULL,
    correct BOOLEAN NOT NULL,
    difficulty INT NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Curriculum snapshot per user
CREATE TABLE IF NOT EXISTS curriculum (
    user_id uuid PRIMARY KEY,
    curriculum_json JSONB NOT NULL
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_question_bank_concepts ON question_bank USING gin (concepts);
CREATE INDEX IF NOT EXISTS idx_user_progress_user_time ON user_progress (user_id, timestamp DESC);

-- HNSW indexes for vector similarity search (works on empty tables, better performance than ivfflat)
CREATE INDEX IF NOT EXISTS idx_question_bank_embedding ON question_bank USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_concept_mastery_embedding ON concept_mastery USING hnsw (embedding vector_cosine_ops);
