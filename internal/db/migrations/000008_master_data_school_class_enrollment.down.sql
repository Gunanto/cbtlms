BEGIN;

DROP INDEX IF EXISTS idx_enrollments_school_class;
DROP TABLE IF EXISTS enrollments;

DROP INDEX IF EXISTS idx_classes_school_grade;
DROP TABLE IF EXISTS classes;

DROP INDEX IF EXISTS ux_schools_code_not_null;
DROP INDEX IF EXISTS ux_schools_name;
DROP TABLE IF EXISTS schools;

COMMIT;
