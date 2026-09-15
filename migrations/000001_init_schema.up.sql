-- ============================================================================
-- FOOD RESCUE - Surplus Food Marketplace & Waste Redistribution Platform
-- Database Schema (MySQL 8.0+ / MariaDB 10.3+)
-- ============================================================================

SET FOREIGN_KEY_CHECKS = 0;

-- 1. USERS
DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    email               VARCHAR(255)    NOT NULL UNIQUE,
    full_name           VARCHAR(150)    NOT NULL,
    photo_url           VARCHAR(500)    NULL,
    phone_number        VARCHAR(20)     NULL,
    auth_provider       ENUM('google', 'email_password') NOT NULL DEFAULT 'email_password',
    role                ENUM('user', 'toko', 'kurir', 'admin') NOT NULL,
    is_ngo_verified     BOOLEAN         NOT NULL DEFAULT FALSE,
    trust_score         DECIMAL(3,2)    NOT NULL DEFAULT 5.00,
    latitude            DECIMAL(10,7)   NULL,
    longitude           DECIMAL(10,7)   NULL,
    address_text        VARCHAR(255)    NULL,
    account_status      ENUM('pending_verification', 'active', 'rejected', 'suspended', 'deactivated') NOT NULL DEFAULT 'active',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_users_role (role),
    INDEX idx_users_status (account_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. PAYMENT METHODS
DROP TABLE IF EXISTS payment_methods;
CREATE TABLE payment_methods (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    user_id             CHAR(36)        NOT NULL,
    provider            VARCHAR(50)     NOT NULL,
    account_reference   VARCHAR(255)    NOT NULL,
    is_default          BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_paymethod_user (user_id),
    CONSTRAINT fk_paymethod_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. TOKO PROFILES
DROP TABLE IF EXISTS toko_profiles;
CREATE TABLE toko_profiles (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    user_id             CHAR(36)        NOT NULL UNIQUE,
    business_name       VARCHAR(150)    NOT NULL,
    business_category   VARCHAR(100)    NOT NULL,
    address             VARCHAR(255)    NOT NULL,
    latitude            DECIMAL(10,7)   NULL,
    longitude           DECIMAL(10,7)   NULL,
    legal_document_url  VARCHAR(500)    NULL,
    operational_hours   VARCHAR(100)    NULL,
    verification_status ENUM('pending', 'approved', 'rejected') NOT NULL DEFAULT 'pending',
    verified_by_admin_id CHAR(36)       NULL,
    verified_at         DATETIME        NULL,
    average_rating      DECIMAL(3,2)    NOT NULL DEFAULT 0.00,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_toko_status (verification_status),
    INDEX idx_toko_verifier (verified_by_admin_id),
    CONSTRAINT fk_toko_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_toko_verifier FOREIGN KEY (verified_by_admin_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. COURIER PROFILES
DROP TABLE IF EXISTS courier_profiles;
CREATE TABLE courier_profiles (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    user_id             CHAR(36)        NOT NULL UNIQUE,
    vehicle_type        VARCHAR(50)     NOT NULL,
    id_document_url     VARCHAR(500)    NULL,
    verification_status ENUM('pending', 'approved', 'rejected') NOT NULL DEFAULT 'pending',
    verified_by_admin_id CHAR(36)       NULL,
    verified_at         DATETIME        NULL,
    is_online           BOOLEAN         NOT NULL DEFAULT FALSE,
    current_latitude    DECIMAL(10,7)   NULL,
    current_longitude   DECIMAL(10,7)   NULL,
    last_location_update DATETIME       NULL,
    average_rating      DECIMAL(3,2)    NOT NULL DEFAULT 0.00,
    total_earnings      DECIMAL(14,2)   NOT NULL DEFAULT 0.00,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_courier_status (verification_status),
    INDEX idx_courier_online (is_online),
    INDEX idx_courier_verifier (verified_by_admin_id),
    CONSTRAINT fk_courier_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_courier_verifier FOREIGN KEY (verified_by_admin_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. FOOD LISTINGS
DROP TABLE IF EXISTS food_listings;
CREATE TABLE food_listings (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    toko_id             CHAR(36)        NOT NULL,
    name                VARCHAR(150)    NOT NULL,
    category            VARCHAR(100)    NOT NULL,
    description         TEXT            NULL,
    photo_url           VARCHAR(500)    NULL,
    initial_price       DECIMAL(12,2)   NOT NULL,
    minimum_price       DECIMAL(12,2)   NOT NULL,
    current_price       DECIMAL(12,2)   NOT NULL,
    stock_quantity      INT             NOT NULL DEFAULT 1,
    food_safety_notes   TEXT            NULL,
    safe_until          DATETIME        NOT NULL,
    pickup_start_time   DATETIME        NULL,
    pickup_end_time     DATETIME        NULL,
    status              ENUM('active', 'sold_out', 'expired', 'pushed_to_community', 'discarded', 'inactive') NOT NULL DEFAULT 'active',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_listing_status (status),
    INDEX idx_listing_safe_until (safe_until),
    INDEX idx_listing_location (toko_id, status),
    CONSTRAINT fk_listing_toko FOREIGN KEY (toko_id) REFERENCES toko_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. ORDERS
DROP TABLE IF EXISTS orders;
CREATE TABLE orders (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    listing_id          CHAR(36)        NOT NULL,
    user_id             CHAR(36)        NOT NULL,
    quantity            INT             NOT NULL DEFAULT 1,
    price_at_purchase   DECIMAL(12,2)   NOT NULL,
    total_amount        DECIMAL(12,2)   NOT NULL,
    fulfillment_method  ENUM('pickup_mandiri', 'diantar_kurir') NOT NULL,
    payment_method      VARCHAR(50)     NULL,
    payment_status      ENUM('unpaid', 'paid', 'refunded', 'failed') NOT NULL DEFAULT 'unpaid',
    order_status        ENUM('menunggu_pickup', 'diantar', 'selesai', 'dibatalkan') NOT NULL DEFAULT 'menunggu_pickup',
    confirmation_code   VARCHAR(10)     NOT NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME        NULL,
    INDEX idx_order_status (order_status),
    INDEX idx_order_user (user_id),
    INDEX idx_order_listing (listing_id),
    CONSTRAINT fk_order_listing FOREIGN KEY (listing_id) REFERENCES food_listings(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_order_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. DELIVERIES
DROP TABLE IF EXISTS deliveries;
CREATE TABLE deliveries (
    id                      CHAR(36)    NOT NULL PRIMARY KEY,
    order_id                CHAR(36)    NOT NULL UNIQUE,
    courier_id              CHAR(36)    NULL,
    matching_status         ENUM('mencari_kurir', 'ditawarkan', 'diterima', 'timeout', 'gagal') NOT NULL DEFAULT 'mencari_kurir',
    pickup_latitude         DECIMAL(10,7) NULL,
    pickup_longitude        DECIMAL(10,7) NULL,
    dropoff_latitude        DECIMAL(10,7) NULL,
    dropoff_longitude       DECIMAL(10,7) NULL,
    trip_status             ENUM('menuju_toko', 'barang_diambil', 'menuju_user', 'diterima_user') NULL,
    pickup_confirmation_code  VARCHAR(10) NULL,
    dropoff_confirmation_code VARCHAR(10) NULL,
    offer_expires_at        DATETIME    NULL,
    delivery_fee            DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    created_at              DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_delivery_courier (courier_id),
    INDEX idx_delivery_matching (matching_status),
    CONSTRAINT fk_delivery_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_delivery_courier FOREIGN KEY (courier_id) REFERENCES courier_profiles(id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7B. DELIVERY OFFERS (tawaran ke kandidat kurir)
DROP TABLE IF EXISTS delivery_offers;
CREATE TABLE delivery_offers (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    delivery_id         CHAR(36)        NOT NULL,
    courier_profile_id  CHAR(36)        NOT NULL,
    offered_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at          DATETIME        NOT NULL,
    accepted            BOOLEAN         NOT NULL DEFAULT FALSE,
    accepted_at         DATETIME        NULL,
    status              ENUM('pending', 'expired', 'accepted', 'declined') NOT NULL DEFAULT 'pending',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_deliveryoffer_delivery (delivery_id),
    INDEX idx_deliveryoffer_courier (courier_profile_id),
    CONSTRAINT fk_deliveryoffer_delivery FOREIGN KEY (delivery_id) REFERENCES deliveries(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_deliveryoffer_courier FOREIGN KEY (courier_profile_id) REFERENCES courier_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 8. COMMUNITY POSTS
DROP TABLE IF EXISTS community_posts;
CREATE TABLE community_posts (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    listing_id          CHAR(36)        NOT NULL,
    toko_id             CHAR(36)        NOT NULL,
    target_category     ENUM('kompos', 'pakan_ternak', 'lainnya') NOT NULL,
    transport_fee       DECIMAL(12,2)   NOT NULL DEFAULT 0.00,
    claim_status        ENUM('tersedia', 'diklaim', 'selesai', 'dibatalkan') NOT NULL DEFAULT 'tersedia',
    claimed_by_user_id  CHAR(36)        NULL,
    claimed_at          DATETIME        NULL,
    completed_at        DATETIME        NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_community_status (claim_status),
    INDEX idx_community_listing (listing_id),
    INDEX idx_community_toko (toko_id),
    INDEX idx_community_claimer (claimed_by_user_id),
    CONSTRAINT fk_community_listing FOREIGN KEY (listing_id) REFERENCES food_listings(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_community_toko FOREIGN KEY (toko_id) REFERENCES toko_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_community_claimer FOREIGN KEY (claimed_by_user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 9. EMERGENCY ALERTS
DROP TABLE IF EXISTS emergency_alerts;
CREATE TABLE emergency_alerts (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    ngo_user_id         CHAR(36)        NOT NULL,
    need_type           VARCHAR(150)    NOT NULL,
    target_area         VARCHAR(255)    NOT NULL,
    urgency_level       ENUM('rendah', 'sedang', 'tinggi', 'kritis') NOT NULL DEFAULT 'sedang',
    description         TEXT            NULL,
    status              ENUM('aktif', 'selesai', 'dibatalkan') NOT NULL DEFAULT 'aktif',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_alert_status (status),
    INDEX idx_alert_ngo (ngo_user_id),
    CONSTRAINT fk_alert_ngo FOREIGN KEY (ngo_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

DROP TABLE IF EXISTS emergency_alert_responses;
CREATE TABLE emergency_alert_responses (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    alert_id            CHAR(36)        NOT NULL,
    toko_id             CHAR(36)        NOT NULL,
    availability_note   TEXT            NULL,
    response_status     ENUM('menawarkan', 'dikonfirmasi', 'selesai') NOT NULL DEFAULT 'menawarkan',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_alertresp_alert (alert_id),
    INDEX idx_alertresp_toko (toko_id),
    CONSTRAINT fk_alertresp_alert FOREIGN KEY (alert_id) REFERENCES emergency_alerts(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_alertresp_toko FOREIGN KEY (toko_id) REFERENCES toko_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 10. IMPACT REPORTS
DROP TABLE IF EXISTS impact_reports;
CREATE TABLE impact_reports (
    id                      CHAR(36)        NOT NULL PRIMARY KEY,
    owner_user_id           CHAR(36)        NOT NULL,
    period_type             ENUM('bulanan', 'tahunan') NOT NULL,
    period_start            DATE            NOT NULL,
    period_end              DATE            NOT NULL,
    total_food_saved_kg     DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    total_money_amount      DECIMAL(14,2)   NOT NULL DEFAULT 0.00,
    estimated_co2_saved_kg  DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    generated_at            DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_impact_owner (owner_user_id),
    INDEX idx_impact_owner_period (owner_user_id, period_start, period_end),
    CONSTRAINT fk_impact_user FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 11. RECIPES
DROP TABLE IF EXISTS recipes;
CREATE TABLE recipes (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    food_category       VARCHAR(100)    NOT NULL,
    title               VARCHAR(150)    NOT NULL,
    ingredients_text    TEXT            NOT NULL,
    steps_text          TEXT            NOT NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_recipe_category (food_category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 12. AI CONVERSATIONS
DROP TABLE IF EXISTS ai_conversations;
CREATE TABLE ai_conversations (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    user_id             CHAR(36)        NOT NULL,
    listing_id          CHAR(36)        NULL,
    source_type         ENUM('dari_listing', 'deteksi_kamera') NOT NULL,
    photo_url           VARCHAR(500)    NULL,
    detected_food_name  VARCHAR(150)    NULL,
    estimated_calories  DECIMAL(8,2)    NULL,
    estimated_protein_g DECIMAL(8,2)    NULL,
    estimated_carbs_g   DECIMAL(8,2)    NULL,
    estimated_fat_g     DECIMAL(8,2)    NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_aiconv_user (user_id),
    INDEX idx_aiconv_listing (listing_id),
    CONSTRAINT fk_aiconv_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_aiconv_listing FOREIGN KEY (listing_id) REFERENCES food_listings(id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

DROP TABLE IF EXISTS ai_conversation_messages;
CREATE TABLE ai_conversation_messages (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    conversation_id     CHAR(36)        NOT NULL,
    sender              ENUM('user', 'ai') NOT NULL,
    message_text        TEXT            NOT NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_aimsg_conv (conversation_id),
    CONSTRAINT fk_aimsg_conv FOREIGN KEY (conversation_id) REFERENCES ai_conversations(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 13. CHATS / MESSAGES
DROP TABLE IF EXISTS chats;
CREATE TABLE chats (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    order_id            CHAR(36)        NOT NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_chat_order (order_id),
    CONSTRAINT fk_chat_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

DROP TABLE IF EXISTS messages;
CREATE TABLE messages (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    chat_id             CHAR(36)        NOT NULL,
    sender_id           CHAR(36)        NOT NULL,
    message_text        TEXT            NOT NULL,
    is_read             BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_message_chat (chat_id),
    INDEX idx_message_sender (sender_id),
    CONSTRAINT fk_message_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_message_sender FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 14. RATINGS
DROP TABLE IF EXISTS ratings;
CREATE TABLE ratings (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    order_id            CHAR(36)        NOT NULL,
    rater_user_id       CHAR(36)        NOT NULL,
    ratee_user_id       CHAR(36)        NOT NULL,
    rating_target       ENUM('toko', 'kurir', 'user') NOT NULL,
    score               TINYINT         NOT NULL,
    review_text         TEXT            NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_rating_order (order_id),
    INDEX idx_rating_rater (rater_user_id),
    INDEX idx_rating_ratee (ratee_user_id),
    CONSTRAINT fk_rating_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_rating_rater FOREIGN KEY (rater_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_rating_ratee FOREIGN KEY (ratee_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT chk_rating_score CHECK (score BETWEEN 1 AND 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 15. REPORTS
DROP TABLE IF EXISTS reports;
CREATE TABLE reports (
    id                  CHAR(36)        NOT NULL PRIMARY KEY,
    reporter_user_id    CHAR(36)        NOT NULL,
    reported_entity_type ENUM('listing', 'user', 'toko', 'kurir', 'order', 'community_post') NOT NULL,
    reported_entity_id  CHAR(36)        NOT NULL,
    reason              TEXT            NOT NULL,
    report_status       ENUM('menunggu', 'ditinjau', 'ditindak', 'ditolak') NOT NULL DEFAULT 'menunggu',
    reviewed_by_admin_id CHAR(36)       NULL,
    reviewed_at         DATETIME        NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_report_reporter (reporter_user_id),
    INDEX idx_report_reviewer (reviewed_by_admin_id),
    INDEX idx_report_status (report_status),
    CONSTRAINT fk_report_reporter FOREIGN KEY (reporter_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_report_reviewer FOREIGN KEY (reviewed_by_admin_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS = 1;