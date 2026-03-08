CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


/* insert into users (name, email) Values 
    ('alex', 'alex@gmail.com'),
    ('mohamed', 'mohamed@gmail.com' */
on conflict do nothing;


CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

DO $$
BEGIN
    RAISE NOTICE 'Database initialized successfully!';
END $$;