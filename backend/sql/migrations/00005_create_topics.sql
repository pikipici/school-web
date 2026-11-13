-- +goose Up
-- The seven fixed project topics from PLAN.md 2.1.
CREATE TABLE topics (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       text    NOT NULL,
    title      text    NOT NULL,
    sort_order integer NOT NULL,
    CONSTRAINT topics_code_format CHECK (code ~ '^K[1-9]$'),
    CONSTRAINT topics_title_not_empty CHECK (char_length(btrim(title)) BETWEEN 1 AND 160)
);

CREATE UNIQUE INDEX topics_code_key ON topics (code);
CREATE UNIQUE INDEX topics_sort_order_key ON topics (sort_order);

INSERT INTO topics (code, title, sort_order) VALUES
    ('K1', 'Struktur Data (Graf dan Tree)', 1),
    ('K2', 'Ekspresi dan Operasi Logika, Algoritma Perulangan', 2),
    ('K3', 'Blogspot', 3),
    ('K4', 'Isu Keamanan Internet (Privasi, Cyberbullying, Hoaks)', 4),
    ('K5', 'Visualisasi Data', 5),
    ('K6', 'Keamanan Data dan Ancaman Digital (Phishing, Malware)', 6),
    ('K7', 'Pseudocode dan Flowchart', 7);

-- +goose Down
DROP TABLE topics;
