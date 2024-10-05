
-- migration file name should be related to the work done in the file
-- if i am adding a table so the migration file name can be 00002_adding_dishes_table.up.sql

BEGIN;

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    archived_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS active_user ON users(TRIM(LOWER(email))) WHERE archived_at IS NULL;

-- Role Type Enum
CREATE TYPE role_type AS ENUM (
    'admin',
    'sub-admin',
    'user'
);

-- User Roles Table
CREATE TABLE IF NOT EXISTS user_roles (

    -- no need of id here only user_id and role_name is enough

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) NOT NULL,
    role_name role_type NOT NULL,
    created_by UUID REFERENCES users(id), -- Corrected to UUID
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    archived_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_role ON user_roles(user_id, role_name) WHERE archived_at IS NULL;

-- User Session Table
CREATE TABLE IF NOT EXISTS user_session (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- no need of user_role_id in user_session table

    user_role_id UUID REFERENCES user_roles(id) NOT NULL,
    user_id UUID REFERENCES users(id) NOT NULL,  -- Corrected table reference to 'users'

    -- no need of session_token in user_session table

    session_token TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User Address Table
CREATE TABLE IF NOT EXISTS user_address (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) NOT NULL,

    -- no need to check address validation here

    address VARCHAR(30) CHECK (address ~ '^[a-zA-Z0-9\s]*$'),
    state VARCHAR(16) CHECK (state ~ '^[a-zA-Z\s]*$'),
    city VARCHAR(20) CHECK (city ~ '^[a-zA-Z\s]*$'),
    pin_code CHAR(6) CHECK (pin_code ~ '^[0-9]*$'),
    lat DOUBLE PRECISION NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lng DOUBLE PRECISION NOT NULL CHECK (lng BETWEEN -180 AND 180),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    archived_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_address ON user_address(user_id, lat, lng) WHERE archived_at IS NULL;

COMMIT;
