package sqlite

const (
	createTable string = `CREATE TABLE IF NOT EXISTS ddns_domains (
			fqdn TEXT NOT NULL,
			update_time TEXT NOT NULL,
			register_type TEXT NOT NULL,
			ip TEXT NOT NULL,
			active BOOL NOT NULL
	)`

	lastRecords string = `SELECT fqdn, ip, register_type FROM ddns_domains WHERE
			active = true
	`
	insertRecord string = `INSERT INTO ddns_domains
			(fqdn, update_time, register_type, ip, active)
			VALUES(?,?,?,?,?)
	`

	deactivateRecord string = `UPDATE ddns_domains SET active = false
			WHERE fqdn = ? AND register_type = ?
	`
)
