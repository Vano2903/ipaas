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
    `githubRepo` varchar(255),
    `lastCommit` varchar(40),
    `branch` varchar(255),
    `port` varchar(5),
    `externalPort` varchar(5),
    `language` varchar(255),
    `createdAt` datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
    `isPublic` tinyint(1) NOT NULL DEFAULT '0',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `envs` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `applicationID` int(11) NOT NULL,
    `key` varchar(255) NOT NULL,
    `value` text NOT NULL,
    PRIMARY KEY (`id`)
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