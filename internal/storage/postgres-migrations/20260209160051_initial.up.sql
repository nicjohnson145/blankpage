BEGIN;

CREATE TABLE blankpage_book_metadata (
    id            TEXT                      NOT NULL,
    title         TEXT                      NOT NULL,
    author        TEXT                              ,
    series        TEXT                              ,
    series_number NUMERIC                           ,
    upload_ts     TIMESTAMP WITH TIME ZONE  NOT NULL
);

CREATE TABLE blankpage_book_content (
    book_id   TEXT    NOT NULL,
    format    TEXT    NOT NULL,
    content   BYTEA   NOT NULL
);

CREATE TABLE blankpage_shelf (
    id     TEXT   NOT NULL,
    name   TEXT   NOT NULL
);

CREATE TABLE blankpage_shelf_member (
    shelf_id  TEXT   NOT NULL,
    book_id   TEXT   NOT NULL
);

ALTER TABLE
    blankpage_book_metadata
ADD CONSTRAINT
    blankpage_book_metadata_pk
PRIMARY KEY (
    id
);

ALTER TABLE
    blankpage_book_content
ADD CONSTRAINT
    blankpage_book_content_pk
PRIMARY KEY (
    book_id,
    format
);

ALTER TABLE
    blankpage_shelf
ADD CONSTRAINT
    blankpage_shelf_pk
PRIMARY KEY (
    id
);

ALTER TABLE
    blankpage_shelf_member
ADD CONSTRAINT
    blankpage_shelf_member_pk
PRIMARY KEY (
    shelf_id,
    book_id
);

ALTER TABLE
    blankpage_book_content
ADD CONSTRAINT
    book_id_fk
FOREIGN KEY (
    book_id
)
REFERENCES
    blankpage_book_metadata(
        id
    )
ON DELETE
    CASCADE
;

ALTER TABLE
    blankpage_shelf_member
ADD CONSTRAINT
    shelf_id_fk
FOREIGN KEY (
    shelf_id
)
REFERENCES
    blankpage_shelf(
        id
    )
ON DELETE
    CASCADE
;

ALTER TABLE
    blankpage_shelf_member
ADD CONSTRAINT
    book_id_fk
FOREIGN KEY (
    book_id
)
REFERENCES
    blankpage_book_metadata(
        id
    )
ON DELETE
    CASCADE
;

COMMIT;
