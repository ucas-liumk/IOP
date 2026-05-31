-- Library management (图书馆) tables — per tenant.
-- Aggregates: Book (a catalog title with copy counts) and Borrow (a loan record).

CREATE TABLE IF NOT EXISTS book (
    id          UUID PRIMARY KEY,
    isbn        TEXT NOT NULL DEFAULT '',
    title       TEXT NOT NULL,
    author      TEXT NOT NULL DEFAULT '',
    publisher   TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT '',
    total       INT  NOT NULL DEFAULT 0,          -- copies owned
    available   INT  NOT NULL DEFAULT 0,          -- copies currently on the shelf
    cover_url   TEXT NOT NULL DEFAULT '',
    location    TEXT NOT NULL DEFAULT '',          -- shelf location
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS book_category_idx ON book(category);
CREATE INDEX IF NOT EXISTS book_title_idx ON book(title);
CREATE INDEX IF NOT EXISTS book_isbn_idx ON book(isbn);

CREATE TABLE IF NOT EXISTS book_borrow (
    id           UUID PRIMARY KEY,
    book_id      UUID NOT NULL REFERENCES book(id) ON DELETE CASCADE,
    member_id    UUID NOT NULL,                    -- borrower (member id)
    borrowed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    due_at       TIMESTAMPTZ NOT NULL,
    returned_at  TIMESTAMPTZ,
    status       TEXT NOT NULL DEFAULT 'borrowed', -- borrowed / returned / overdue
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS book_borrow_member_idx ON book_borrow(member_id, status);
CREATE INDEX IF NOT EXISTS book_borrow_book_idx ON book_borrow(book_id);
CREATE INDEX IF NOT EXISTS book_borrow_status_idx ON book_borrow(status);
