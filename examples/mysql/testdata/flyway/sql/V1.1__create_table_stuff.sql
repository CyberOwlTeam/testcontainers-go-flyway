CREATE TABLE stuff
(
    id   VARCHAR(255) NOT NULL PRIMARY KEY DEFAULT (UUID()),
    name VARCHAR(255) NOT NULL
);