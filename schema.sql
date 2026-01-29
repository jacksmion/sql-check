CREATE TABLE users (
    id BIGINT AUTO_INCREMENT,
    name VARCHAR(255),
    email VARCHAR(255),
    created_at DATETIME,
    PRIMARY KEY (id),
    KEY idx_email (email)
);

CREATE TABLE orders (
    id BIGINT AUTO_INCREMENT,
    user_id BIGINT,
    amount DECIMAL(10, 2),
    status VARCHAR(50),
    created_at DATETIME,
    PRIMARY KEY (id),
    KEY idx_user_status (user_id, status)
);
