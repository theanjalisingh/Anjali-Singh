-- Seed identity master data and local development admin user.
-- Safe to re-run: inserts only when records are missing.
-- Default admin password: Admin@123 (bcrypt, for local development only).

INSERT INTO identity.role (role_code, role_name, is_active, is_deleted)
SELECT 'SUPER_ADMIN', 'Super Admin', TRUE, FALSE
WHERE NOT EXISTS (
    SELECT 1 FROM identity.role WHERE role_code = 'SUPER_ADMIN' AND is_deleted = FALSE
);

INSERT INTO identity.role (role_code, role_name, is_active, is_deleted)
SELECT 'ORG_ADMIN', 'Org Admin', TRUE, FALSE
WHERE NOT EXISTS (
    SELECT 1 FROM identity.role WHERE role_code = 'ORG_ADMIN' AND is_deleted = FALSE
);

INSERT INTO identity.role (role_code, role_name, is_active, is_deleted)
SELECT 'CLIENT', 'Client', TRUE, FALSE
WHERE NOT EXISTS (
    SELECT 1 FROM identity.role WHERE role_code = 'CLIENT' AND is_deleted = FALSE
);

INSERT INTO identity.organization (organization_name, is_active, is_deleted)
SELECT 'Future Environs', TRUE, FALSE
WHERE NOT EXISTS (
    SELECT 1
    FROM identity.organization
    WHERE organization_name = 'Future Environs'
      AND is_deleted = FALSE
);

DO $$
DECLARE
    v_org_id uuid;
    v_role_id uuid;
    v_user_id uuid;
    v_password_hash text := '$2a$12$nKXUX2MH.KtpPvpoL6kZkebhCI.V6VHQ63CvooNoBspoiGPjD4tJa';
BEGIN
    SELECT organization_id
    INTO v_org_id
    FROM identity.organization
    WHERE organization_name = 'Future Environs'
      AND is_deleted = FALSE
    ORDER BY created_on
    LIMIT 1;

    SELECT role_id
    INTO v_role_id
    FROM identity.role
    WHERE role_code = 'SUPER_ADMIN'
      AND is_deleted = FALSE
    LIMIT 1;

    IF v_org_id IS NULL OR v_role_id IS NULL THEN
        RAISE EXCEPTION 'Seed prerequisites missing: organization or SUPER_ADMIN role';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM identity.users
        WHERE LOWER(email_id) = 'admin@futureenvirons.com'
          AND is_deleted = FALSE
    ) THEN
        RETURN;
    END IF;

    INSERT INTO identity.users (
        user_name,
        contact_no,
        email_id,
        address,
        organization_id,
        valid_till,
        is_active,
        is_deleted,
        is_approved,
        created_on
    )
    VALUES (
        'System Administrator',
        '+91-9999999999',
        'admin@futureenvirons.com',
        'India',
        v_org_id,
        CURRENT_TIMESTAMP + INTERVAL '1 year',
        TRUE,
        FALSE,
        TRUE,
        CURRENT_TIMESTAMP
    )
    RETURNING user_id INTO v_user_id;

    INSERT INTO identity.user_credentials (user_id, password_hash, created_on)
    VALUES (v_user_id, v_password_hash, CURRENT_TIMESTAMP);

    INSERT INTO identity.user_role_assignment (user_id, role_id, created_on, is_active)
    VALUES (v_user_id, v_role_id, CURRENT_TIMESTAMP, TRUE);
END $$;
