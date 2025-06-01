CREATE TABLE IF NOT EXISTS `members` (
    `member_id` int(11) NOT NULL AUTO_INCREMENT,
    `customer_id` varchar(255) NOT NULL,
    `access_id` uuid DEFAULT NULL,
    `name` varchar(255) NOT NULL,
    `email` varchar(255) NOT NULL,
    `status` enum('not_active','active') DEFAULT 'not_active',
    PRIMARY KEY (`member_id`),
    UNIQUE KEY `customer_id` (`customer_id`),
    UNIQUE KEY `email` (`email`),
    UNIQUE KEY `access_id` (`access_id`)
);
