BEGIN;

-- no need of transaction because its a single query and also use if not exists in all query
ALTER TABLE restaurants ADD COLUMN created_by UUID REFERENCES users(id);

COMMIT;
