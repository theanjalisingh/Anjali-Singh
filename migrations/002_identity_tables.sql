-- Identity tables for futureEnvirons (Phase 1)
-- Safe to re-run on fresh or partially-created databases.

CREATE TABLE IF NOT EXISTS identity.role (
    role_id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_code character varying(50) NOT NULL,
    role_name character varying(100) NOT NULL,
    created_by uuid,
    created_on timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    modified_by uuid,
    modified_on timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL
);

CREATE TABLE IF NOT EXISTS identity.organization (
    organization_id uuid DEFAULT gen_random_uuid() NOT NULL,
    organization_name character varying(200) NOT NULL,
    parent_organization_id uuid,
    org_logo_url text,
    is_active boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    created_by uuid,
    created_on timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    modified_by uuid,
    modified_on timestamp with time zone
);

CREATE TABLE IF NOT EXISTS identity.users (
    user_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_name character varying(150) NOT NULL,
    contact_no character varying(30),
    email_id character varying(255) NOT NULL,
    address text,
    organization_id uuid,
    valid_till timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    is_approved boolean DEFAULT false NOT NULL,
    created_by uuid,
    created_on timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    modified_by uuid,
    modified_on timestamp with time zone
);

CREATE TABLE IF NOT EXISTS identity.user_credentials (
    user_id uuid NOT NULL,
    password_hash text NOT NULL,
    password_changed_on timestamp with time zone,
    failed_login_count integer DEFAULT 0 NOT NULL,
    locked_until timestamp with time zone,
    last_login_on timestamp with time zone,
    created_on timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    modified_on timestamp with time zone
);

CREATE TABLE IF NOT EXISTS identity.user_role_assignment (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    created_by uuid,
    created_on timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    modified_by uuid,
    modified_on timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'role_pkey' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.role ADD CONSTRAINT role_pkey PRIMARY KEY (role_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_role_code' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.role ADD CONSTRAINT uq_role_code UNIQUE (role_code);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'organization_pkey' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.organization ADD CONSTRAINT organization_pkey PRIMARY KEY (organization_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_pkey' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.users ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'user_credentials_pkey' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.user_credentials ADD CONSTRAINT user_credentials_pkey PRIMARY KEY (user_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'pk_user_role_assignment' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.user_role_assignment ADD CONSTRAINT pk_user_role_assignment PRIMARY KEY (user_id, role_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_organization_parent' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.organization
            ADD CONSTRAINT fk_organization_parent
            FOREIGN KEY (parent_organization_id) REFERENCES identity.organization(organization_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_user_organization' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.users
            ADD CONSTRAINT fk_user_organization
            FOREIGN KEY (organization_id) REFERENCES identity.organization(organization_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_user_credentials_user' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.user_credentials
            ADD CONSTRAINT fk_user_credentials_user
            FOREIGN KEY (user_id) REFERENCES identity.users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_ura_user' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.user_role_assignment
            ADD CONSTRAINT fk_ura_user
            FOREIGN KEY (user_id) REFERENCES identity.users(user_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_ura_role' AND connamespace = 'identity'::regnamespace
    ) THEN
        ALTER TABLE identity.user_role_assignment
            ADD CONSTRAINT fk_ura_role
            FOREIGN KEY (role_id) REFERENCES identity.role(role_id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_organization_parent
    ON identity.organization USING btree (parent_organization_id)
    WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_users_organization
    ON identity.users USING btree (organization_id)
    WHERE is_deleted = false;

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email
    ON identity.users USING btree (lower(email_id::text))
    WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_user_role_assignment_user
    ON identity.user_role_assignment USING btree (user_id)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_user_role_assignment_role
    ON identity.user_role_assignment USING btree (role_id)
    WHERE is_active = true;
