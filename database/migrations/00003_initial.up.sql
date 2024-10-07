BEGIN;

-- try to add if not exists before creating column or table
-- there is no need of begin and commit because you are executing single statement

ALTER TABLE restaurants ADD COLUMN created_by UUID REFERENCES users(id);

COMMIT;
