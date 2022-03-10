CREATE DATABASE IF NOT EXISTS ipaas;

USE ipaas;

CREATE TABLE IF NOT EXISTS `states` (
    `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `state` char(24) NOT NULL,
    `issDate` DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `tokens` (
    `userID` int not null,
    `accToken` char(64) NOT NULL,
    `accExp` DATETIME NOT NULL,
    `refreshToken` char(64) NOT NULL,
    `refreshExp` DATETIME NOT NULL,
    PRIMARY KEY (`userID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `users`(
    `userID` int not null,
    `name` varchar(30) NOT NULL,
    `lastName` varchar(30) NOT NULL,
    `email` varchar(75) NOT NULL,
    `pfp` varchar(55) NOT NULL,
    PRIMARY KEY(`userID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
