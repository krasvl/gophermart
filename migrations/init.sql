CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	login VARCHAR(50) NOT NULL UNIQUE,
	password VARCHAR(60) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS balances (
	user_id INT PRIMARY KEY,
	current FLOAT NOT NULL,
	withdrawn FLOAT NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS withdrawals (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL,
	order_number VARCHAR(50) NOT NULL,
	sum FLOAT NOT NULL,
	processed_at TIMESTAMP NOT NULL DEFAULT now(),
	FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS orders (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL,
	number VARCHAR(50) NOT NULL UNIQUE,
	status VARCHAR(50) NOT NULL CHECK (status IN ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')),
	accrual FLOAT NOT NULL,
	uploaded_at TIMESTAMP NOT NULL DEFAULT now(),
	FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE OR REPLACE FUNCTION create_default_balance()
RETURNS TRIGGER AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM balances WHERE user_id = NEW.id) THEN
		INSERT INTO balances (user_id, current, withdrawn) VALUES (NEW.id, 0, 0);
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_default_balance
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION create_default_balance();
