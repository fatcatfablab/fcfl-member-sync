CREATE TABLE IF NOT EXISTS `customers` (
    `customer_id` VARCHAR(255),
    `name` VARCHAR(255) NOT NULL,
    `email` VARCHAR(255) NOT NULL,
    `delinquent` BOOLEAN DEFAULT(0),
    PRIMARY KEY (`customer_id`),
    UNIQUE KEY (`email`)
);
