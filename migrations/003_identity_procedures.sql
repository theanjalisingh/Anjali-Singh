-- Identity functions and stored procedures (Phase 1)
-- Safe to re-run: uses CREATE OR REPLACE.

CREATE OR REPLACE FUNCTION identity.fn_get_user_for_login(p_email_id character varying)
RETURNS TABLE(
    user_id uuid,
    user_name character varying,
    email_id character varying,
    organization_id uuid,
    organization_name character varying,
    password_hash text,
    is_active boolean,
    is_approved boolean,
    valid_till timestamp with time zone,
    role_id uuid,
    role_code character varying,
    role_name character varying
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT
        u.user_id,
        u.user_name,
        u.email_id,
        u.organization_id,
        o.organization_name,
        uc.password_hash,
        u.is_active,
        u.is_approved,
        u.valid_till,
        r.role_id,
        r.role_code,
        r.role_name
    FROM identity.users u
    INNER JOIN identity.user_credentials uc
        ON uc.user_id = u.user_id
    LEFT JOIN identity.organization o
        ON o.organization_id = u.organization_id
    LEFT JOIN identity.user_role_assignment ura
        ON ura.user_id = u.user_id
        AND ura.is_active = TRUE
    LEFT JOIN identity.role r
        ON r.role_id = ura.role_id
        AND r.is_active = TRUE
        AND r.is_deleted = FALSE
    WHERE LOWER(u.email_id) = LOWER(p_email_id)
      AND u.is_deleted = FALSE;
END;
$$;

CREATE OR REPLACE PROCEDURE identity.sp_get_login_details(
    IN p_email_id character varying,
    INOUT v_rc refcursor,
    IN p_optype integer DEFAULT 1
)
LANGUAGE plpgsql
AS $$
BEGIN
    IF p_optype = 1 THEN
        OPEN v_rc FOR
        SELECT
            u.user_id,
            u.user_name,
            u.email_id,
            u.contact_no,
            u.organization_id,
            o.organization_name,
            u.is_active,
            u.is_approved,
            u.valid_till,
            uc.password_hash,
            r.role_id,
            r.role_code,
            r.role_name
        FROM identity.users u
        INNER JOIN identity.user_credentials uc
            ON uc.user_id = u.user_id
        LEFT JOIN identity.organization o
            ON o.organization_id = u.organization_id
            AND o.is_deleted = FALSE
        LEFT JOIN identity.user_role_assignment ura
            ON ura.user_id = u.user_id
            AND ura.is_active = TRUE
        LEFT JOIN identity.role r
            ON r.role_id = ura.role_id
            AND r.is_active = TRUE
            AND r.is_deleted = FALSE
        WHERE LOWER(u.email_id) = LOWER(p_email_id)
          AND u.is_deleted = FALSE;
    END IF;
END;
$$;

CREATE OR REPLACE PROCEDURE identity.sp_manage_user(
    IN p_user_id uuid DEFAULT NULL::uuid,
    IN p_user_name character varying DEFAULT NULL::character varying,
    IN p_contact_no character varying DEFAULT NULL::character varying,
    IN p_email_id character varying DEFAULT NULL::character varying,
    IN p_address text DEFAULT NULL::text,
    IN p_organization_id uuid DEFAULT NULL::uuid,
    IN p_valid_till timestamp with time zone DEFAULT NULL::timestamp with time zone,
    IN p_is_active boolean DEFAULT true,
    IN p_is_approved boolean DEFAULT false,
    IN p_is_deleted boolean DEFAULT false,
    IN p_created_by uuid DEFAULT NULL::uuid,
    IN p_optype integer DEFAULT 1,
    INOUT v_rc refcursor DEFAULT NULL::refcursor
)
LANGUAGE plpgsql
AS $$
BEGIN
    /*
        OPERATION TYPES
        1 = Get User By ID
        2 = Insert User
        3 = Update User
        4 = Delete User (Soft Delete)
        5 = Get All Users
    */

    IF p_optype = 1 THEN
        OPEN v_rc FOR
        SELECT
            u.user_id,
            u.user_name,
            u.contact_no,
            u.email_id,
            u.address,
            u.organization_id,
            o.organization_name,
            u.valid_till,
            u.is_active,
            u.is_deleted,
            u.is_approved,
            u.created_by,
            u.created_on,
            u.modified_by,
            u.modified_on
        FROM identity.users u
        LEFT JOIN identity.organization o
            ON o.organization_id = u.organization_id
            AND o.is_deleted = FALSE
        WHERE u.user_id = p_user_id
          AND u.is_deleted = FALSE;

    ELSIF p_optype = 2 THEN
        IF EXISTS (
            SELECT 1
            FROM identity.users
            WHERE LOWER(email_id) = LOWER(p_email_id)
              AND is_deleted = FALSE
        ) THEN
            RAISE EXCEPTION 'User with email % already exists', p_email_id;
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
            created_by,
            created_on
        )
        VALUES (
            p_user_name,
            p_contact_no,
            p_email_id,
            p_address,
            p_organization_id,
            p_valid_till,
            p_is_active,
            FALSE,
            p_is_approved,
            p_created_by,
            CURRENT_TIMESTAMP
        );

        OPEN v_rc FOR
        SELECT
            u.user_id,
            u.user_name,
            u.contact_no,
            u.email_id,
            u.address,
            u.organization_id,
            o.organization_name,
            u.valid_till,
            u.is_active,
            u.is_deleted,
            u.is_approved,
            u.created_by,
            u.created_on
        FROM identity.users u
        LEFT JOIN identity.organization o
            ON o.organization_id = u.organization_id
        WHERE LOWER(u.email_id) = LOWER(p_email_id)
          AND u.is_deleted = FALSE;

    ELSIF p_optype = 3 THEN
        UPDATE identity.users
        SET
            user_name = COALESCE(p_user_name, user_name),
            contact_no = COALESCE(p_contact_no, contact_no),
            email_id = COALESCE(p_email_id, email_id),
            address = COALESCE(p_address, address),
            organization_id = COALESCE(p_organization_id, organization_id),
            valid_till = p_valid_till,
            is_active = p_is_active,
            is_approved = p_is_approved,
            modified_by = p_created_by,
            modified_on = CURRENT_TIMESTAMP
        WHERE user_id = p_user_id
          AND is_deleted = FALSE;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'User not found: %', p_user_id;
        END IF;

        OPEN v_rc FOR
        SELECT
            u.user_id,
            u.user_name,
            u.contact_no,
            u.email_id,
            u.address,
            u.organization_id,
            o.organization_name,
            u.valid_till,
            u.is_active,
            u.is_deleted,
            u.is_approved,
            u.modified_by,
            u.modified_on
        FROM identity.users u
        LEFT JOIN identity.organization o
            ON o.organization_id = u.organization_id
        WHERE u.user_id = p_user_id;

    ELSIF p_optype = 4 THEN
        UPDATE identity.users
        SET
            is_deleted = TRUE,
            is_active = FALSE,
            modified_by = p_created_by,
            modified_on = CURRENT_TIMESTAMP
        WHERE user_id = p_user_id
          AND is_deleted = FALSE;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'User not found: %', p_user_id;
        END IF;

        OPEN v_rc FOR
        SELECT
            user_id,
            user_name,
            email_id,
            is_active,
            is_deleted,
            modified_by,
            modified_on
        FROM identity.users
        WHERE user_id = p_user_id;

    ELSIF p_optype = 5 THEN
        OPEN v_rc FOR
        SELECT
            u.user_id,
            u.user_name,
            u.contact_no,
            u.email_id,
            u.address,
            u.organization_id,
            o.organization_name,
            u.valid_till,
            u.is_active,
            u.is_deleted,
            u.is_approved,
            u.created_by,
            u.created_on,
            u.modified_by,
            u.modified_on
        FROM identity.users u
        LEFT JOIN identity.organization o
            ON o.organization_id = u.organization_id
            AND o.is_deleted = FALSE
        WHERE u.is_deleted = FALSE
        ORDER BY u.created_on DESC;

    ELSE
        RAISE EXCEPTION 'Invalid operation type: %', p_optype;
    END IF;
END;
$$;
