CREATE TABLE IF NOT EXISTS `members` (
    `member_id` INT AUTO_INCREMENT,
    `customer_id` VARCHAR(255) NOT NULL,
    `access_id` UUID,
    PRIMARY KEY (`member_id`),
    UNIQUE KEY (`access_id`),
    FOREIGN KEY (`customer_id`) REFERENCES `customers` (`customer_id`)
);
