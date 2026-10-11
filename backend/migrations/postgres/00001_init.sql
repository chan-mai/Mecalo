-- +goose Up
CREATE TABLE users (
	id           TEXT PRIMARY KEY,
	display_name TEXT NOT NULL,
	role         TEXT NOT NULL,
	created_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE credentials (
	provider      TEXT NOT NULL,
	subject       TEXT NOT NULL,
	user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	password_hash TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (provider, subject)
);
CREATE INDEX credentials_user_idx ON credentials (user_id);

CREATE TABLE sessions (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

CREATE TABLE datasets (
	id          TEXT PRIMARY KEY,
	kind        TEXT NOT NULL,
	name        TEXT NOT NULL,
	version     TEXT NOT NULL DEFAULT '',
	license     TEXT NOT NULL DEFAULT '',
	url         TEXT NOT NULL DEFAULT '',
	attribution TEXT NOT NULL DEFAULT '',
	imported_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE foods (
	id                  TEXT PRIMARY KEY,
	owner_id            TEXT REFERENCES users (id) ON DELETE CASCADE,
	kind                TEXT NOT NULL,
	barcode             TEXT NOT NULL DEFAULT '',
	current_revision_id TEXT NOT NULL,
	created_at          TIMESTAMPTZ NOT NULL,
	deleted_at          TIMESTAMPTZ
);
CREATE INDEX foods_owner_idx ON foods (owner_id);
CREATE INDEX foods_barcode_idx ON foods (barcode) WHERE barcode <> '';

CREATE TABLE food_revisions (
	id         TEXT PRIMARY KEY,
	food_id    TEXT NOT NULL REFERENCES foods (id) ON DELETE CASCADE,
	number     INT NOT NULL,
	name       TEXT NOT NULL,
	brand      TEXT NOT NULL DEFAULT '',
	search_key TEXT NOT NULL,
	dataset_id TEXT NOT NULL REFERENCES datasets (id),
	source_ref TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL,
	UNIQUE (food_id, number)
);
CREATE INDEX food_revisions_source_idx ON food_revisions (dataset_id, source_ref);
CREATE INVERTED INDEX food_revisions_search_idx ON food_revisions (search_key gin_trgm_ops);

CREATE TABLE food_nutrients (
	revision_id TEXT NOT NULL REFERENCES food_revisions (id) ON DELETE CASCADE,
	code        TEXT NOT NULL,
	amount      NUMERIC NOT NULL,
	status      TEXT NOT NULL,
	coverage    NUMERIC NOT NULL,
	PRIMARY KEY (revision_id, code)
);

CREATE TABLE food_portions (
	revision_id TEXT NOT NULL REFERENCES food_revisions (id) ON DELETE CASCADE,
	position    INT NOT NULL,
	unit        TEXT NOT NULL,
	grams       NUMERIC NOT NULL,
	PRIMARY KEY (revision_id, unit)
);

CREATE TABLE recipe_details (
	revision_id  TEXT PRIMARY KEY REFERENCES food_revisions (id) ON DELETE CASCADE,
	raw_grams    NUMERIC NOT NULL,
	cooked_grams NUMERIC NOT NULL,
	servings     NUMERIC NOT NULL
);

-- 材料は参照先の削除後も記録として残す
CREATE TABLE recipe_ingredients (
	revision_id            TEXT NOT NULL REFERENCES recipe_details (revision_id) ON DELETE CASCADE,
	position               INT NOT NULL,
	food_id                TEXT NOT NULL,
	ingredient_revision_id TEXT NOT NULL,
	name                   TEXT NOT NULL,
	quantity_amount        NUMERIC NOT NULL,
	quantity_unit          TEXT NOT NULL,
	grams                  NUMERIC NOT NULL,
	PRIMARY KEY (revision_id, position)
);

-- 食品テーブルへの外部キーなし。スナップショットで完結
CREATE TABLE meal_entries (
	id               TEXT PRIMARY KEY,
	user_id          TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	date             DATE NOT NULL,
	slot             TEXT NOT NULL,
	food_id          TEXT NOT NULL,
	food_revision_id TEXT NOT NULL,
	food_name        TEXT NOT NULL,
	brand            TEXT NOT NULL DEFAULT '',
	dataset_id       TEXT NOT NULL DEFAULT '',
	source_ref       TEXT NOT NULL DEFAULT '',
	quantity_amount  NUMERIC NOT NULL,
	quantity_unit    TEXT NOT NULL,
	grams            NUMERIC NOT NULL,
	client_ref       TEXT NOT NULL DEFAULT '',
	created_at       TIMESTAMPTZ NOT NULL,
	updated_at       TIMESTAMPTZ NOT NULL
);
CREATE INDEX meal_entries_user_date_idx ON meal_entries (user_id, date);
CREATE INDEX meal_entries_user_created_idx ON meal_entries (user_id, created_at);
CREATE UNIQUE INDEX meal_entries_client_ref_idx ON meal_entries (user_id, client_ref) WHERE client_ref <> '';

CREATE TABLE meal_entry_nutrients (
	entry_id TEXT NOT NULL REFERENCES meal_entries (id) ON DELETE CASCADE,
	code     TEXT NOT NULL,
	amount   NUMERIC NOT NULL,
	status   TEXT NOT NULL,
	coverage NUMERIC NOT NULL,
	PRIMARY KEY (entry_id, code)
);

CREATE TABLE favorites (
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	food_id    TEXT NOT NULL REFERENCES foods (id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (user_id, food_id)
);

CREATE TABLE search_history (
	user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	query       TEXT NOT NULL,
	searched_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (user_id, query)
);

CREATE TABLE settings (
	user_id            TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	effective_from     DATE NOT NULL,
	sex                TEXT NOT NULL,
	birth_date         DATE NOT NULL,
	height_cm          NUMERIC NOT NULL,
	activity           TEXT NOT NULL,
	goal               TEXT NOT NULL,
	energy_adjust_kcal NUMERIC NOT NULL,
	menstruating       BOOL NOT NULL,
	updated_at         TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (user_id, effective_from)
);

CREATE TABLE setting_overrides (
	user_id        TEXT NOT NULL,
	effective_from DATE NOT NULL,
	code           TEXT NOT NULL,
	min_amount     NUMERIC,
	max_amount     NUMERIC,
	PRIMARY KEY (user_id, effective_from, code),
	FOREIGN KEY (user_id, effective_from) REFERENCES settings (user_id, effective_from) ON DELETE CASCADE
);

CREATE TABLE weights (
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	date         DATE NOT NULL,
	kg           NUMERIC NOT NULL,
	body_fat_pct NUMERIC,
	PRIMARY KEY (user_id, date)
);

-- +goose Down
DROP TABLE weights;
DROP TABLE setting_overrides;
DROP TABLE settings;
DROP TABLE search_history;
DROP TABLE favorites;
DROP TABLE meal_entry_nutrients;
DROP TABLE meal_entries;
DROP TABLE recipe_ingredients;
DROP TABLE recipe_details;
DROP TABLE food_portions;
DROP TABLE food_nutrients;
DROP TABLE food_revisions;
DROP TABLE foods;
DROP TABLE datasets;
DROP TABLE sessions;
DROP TABLE credentials;
DROP TABLE users;
