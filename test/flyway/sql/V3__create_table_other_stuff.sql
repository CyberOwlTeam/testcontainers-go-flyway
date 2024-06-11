CREATE TABLE other_stuff
(
    id   UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    stuff_id UUID NOT NULL,
    CONSTRAINT fk_stuff_id2stuff
        FOREIGN KEY (stuff_id)
            REFERENCES stuff (id) MATCH FULL
);
