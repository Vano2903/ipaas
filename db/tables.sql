CREATE DATABASE IF NOT EXISTS ipaas;

USE ipaas;

CREATE TABLE IF NOT EXISTS `applications` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `containerID` char(64) NOT NULL,
    `status` varchar(255) NOT NULL,
    `studentID` int(11) NOT NULL,
    `type` varchar(255) NOT NULL,
    `name` varchar(255) NOT NULL,
    `description` TEXT NOT NULL,
    `createdAt` datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
    `isPulic` tinyint(1) NOT NULL DEFAULT '0',
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `states` (
    `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `state` char(24) NOT NULL,
    `issDate` DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `users`(
    `userID` int not null,
    `name` varchar(30) NOT NULL,
    `lastName` varchar(30) NOT NULL,
    `email` varchar(75) NOT NULL,
    `pfp` varchar(55) NOT NULL,
    PRIMARY KEY(`userID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `tokens` (
    `ID` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `userID` int not null,
    `accId` int NOT NULL,
    `refID` int NOT NULL,
    PRIMARY KEY (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `accessTokens` (
    `ID` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `accToken` char(64) NOT NULL,
    `accExp` DATETIME NOT NULL,
    PRIMARY KEY (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `refreshTokens` (
    `ID` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `refreshToken` char(64) NOT NULL,
    `refreshExp` DATETIME NOT NULL,
    PRIMARY KEY (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

