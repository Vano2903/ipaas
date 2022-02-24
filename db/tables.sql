CREATE TABLE IF NOT EXISTS `states` (
    `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
    `state` char(24) NOT NULL,
    `issDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;