package store

import (
	"database/sql"
	"example.com/appupdatemanager/server/internal/model"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB 是对标准 sql.DB 的封装，提供 SQLite 数据库访问能力。
type DB struct {
	*sql.DB
}

// Open 打开或创建指定数据目录下的 SQLite 数据库，并启用 WAL 模式。
func Open(dataDir string) (*DB, error) {
	if err := ensureDir(dataDir); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "app.db")
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	return &DB{sqlDB}, nil
}

// Migrate 执行数据库迁移，创建所有必需的表（若不存在）。
func Migrate(db *DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS software_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    filename TEXT NOT NULL,
    filepath TEXT NOT NULL,
    is_latest INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS client_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL,
    filename TEXT NOT NULL,
    filepath TEXT NOT NULL,
    is_latest INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS resource_packages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    filename TEXT NOT NULL,
    filepath TEXT NOT NULL,
    is_latest INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS clients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    client_version TEXT,
    software_version TEXT,
    status TEXT,
    is_running INTEGER DEFAULT 0,
    ip TEXT,
    os_version TEXT,
    memory TEXT,
    cpu TEXT,
    process_runtime INTEGER DEFAULT 0,
    last_seen DATETIME,
    online_since DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS client_commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER NOT NULL,
    command_type TEXT NOT NULL,
    payload TEXT,
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := db.Exec(schema)
	return err
}

// ensureDir 确保指定目录存在，不存在则创建。
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// UpsertUser 根据用户名插入或更新用户记录，冲突时更新密码哈希。
func UpsertUser(db *DB, user *model.User) error {
	_, err := db.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)
		 ON CONFLICT(username) DO UPDATE SET password_hash=excluded.password_hash`,
		user.Username, user.PasswordHash,
	)
	return err
}

// GetUserByUsername 根据用户名查询用户。
func GetUserByUsername(db *DB, username string) (*model.User, error) {
	row := db.QueryRow(`SELECT id, username, password_hash FROM users WHERE username = ?`, username)
	u := &model.User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// --- Software Versions ---

// ListSoftwareVersions 按创建时间降序返回所有软件版本。
func ListSoftwareVersions(db *DB) ([]model.SoftwareVersion, error) {
	rows, err := db.Query(`SELECT id, name, version, filename, filepath, is_latest, created_at FROM software_versions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.SoftwareVersion, 0)
	for rows.Next() {
		var v model.SoftwareVersion
		var isLatest int
		if err := rows.Scan(&v.ID, &v.Name, &v.Version, &v.Filename, &v.Filepath, &isLatest, &v.CreatedAt); err != nil {
			return nil, err
		}
		v.IsLatest = isLatest == 1
		list = append(list, v)
	}
	return list, rows.Err()
}

// CreateSoftwareVersion 插入一条新的软件版本记录，并回填自增 ID。
func CreateSoftwareVersion(db *DB, v *model.SoftwareVersion) error {
	res, err := db.Exec(
		`INSERT INTO software_versions (name, version, filename, filepath, is_latest) VALUES (?, ?, ?, ?, ?)`,
		v.Name, v.Version, v.Filename, v.Filepath, boolToInt(v.IsLatest),
	)
	if err != nil {
		return err
	}
	v.ID, _ = res.LastInsertId()
	return nil
}

// DeleteSoftwareVersion 删除指定 id 的软件版本记录，并返回其文件路径。
func DeleteSoftwareVersion(db *DB, id int64) (string, error) {
	row := db.QueryRow(`SELECT filepath FROM software_versions WHERE id = ?`, id)
	var path string
	if err := row.Scan(&path); err != nil {
		return "", err
	}
	_, err := db.Exec(`DELETE FROM software_versions WHERE id = ?`, id)
	return path, err
}

// SetLatestSoftwareVersion 清除所有软件版本的最新标记，并将指定 id 设为最新。
func SetLatestSoftwareVersion(db *DB, id int64) error {
	_, err := db.Exec(`UPDATE software_versions SET is_latest = 0`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE software_versions SET is_latest = 1 WHERE id = ?`, id)
	return err
}

// GetLatestSoftwareVersion 返回标记为最新的软件版本。
func GetLatestSoftwareVersion(db *DB) (*model.SoftwareVersion, error) {
	row := db.QueryRow(`SELECT id, name, version, filename, filepath, is_latest, created_at FROM software_versions WHERE is_latest = 1 LIMIT 1`)
	var v model.SoftwareVersion
	var isLatest int
	err := row.Scan(&v.ID, &v.Name, &v.Version, &v.Filename, &v.Filepath, &isLatest, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	v.IsLatest = isLatest == 1
	return &v, nil
}

// UpdateSoftwareName 更新指定软件版本的显示名称。
func UpdateSoftwareName(db *DB, id int64, name string) error {
	_, err := db.Exec(`UPDATE software_versions SET name = ? WHERE id = ?`, name, id)
	return err
}

// --- Client Versions ---

// ListClientVersions 按创建时间降序返回所有客户端版本。
func ListClientVersions(db *DB) ([]model.ClientVersion, error) {
	rows, err := db.Query(`SELECT id, version, filename, filepath, is_latest, created_at FROM client_versions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.ClientVersion, 0)
	for rows.Next() {
		var v model.ClientVersion
		var isLatest int
		if err := rows.Scan(&v.ID, &v.Version, &v.Filename, &v.Filepath, &isLatest, &v.CreatedAt); err != nil {
			return nil, err
		}
		v.IsLatest = isLatest == 1
		list = append(list, v)
	}
	return list, rows.Err()
}

// CreateClientVersion 插入一条新的客户端版本记录，并回填自增 ID。
func CreateClientVersion(db *DB, v *model.ClientVersion) error {
	res, err := db.Exec(
		`INSERT INTO client_versions (version, filename, filepath, is_latest) VALUES (?, ?, ?, ?)`,
		v.Version, v.Filename, v.Filepath, boolToInt(v.IsLatest),
	)
	if err != nil {
		return err
	}
	v.ID, _ = res.LastInsertId()
	return nil
}

// SetLatestClientVersion 清除所有客户端版本的最新标记，并将指定 id 设为最新。
func SetLatestClientVersion(db *DB, id int64) error {
	_, err := db.Exec(`UPDATE client_versions SET is_latest = 0`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE client_versions SET is_latest = 1 WHERE id = ?`, id)
	return err
}

// GetLatestClientVersion 返回标记为最新的客户端版本。
func GetLatestClientVersion(db *DB) (*model.ClientVersion, error) {
	row := db.QueryRow(`SELECT id, version, filename, filepath, is_latest, created_at FROM client_versions WHERE is_latest = 1 LIMIT 1`)
	var v model.ClientVersion
	var isLatest int
	err := row.Scan(&v.ID, &v.Version, &v.Filename, &v.Filepath, &isLatest, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	v.IsLatest = isLatest == 1
	return &v, nil
}

// DeleteClientVersion 删除指定 id 的客户端版本记录，并返回其文件路径。
func DeleteClientVersion(db *DB, id int64) (string, error) {
	row := db.QueryRow(`SELECT filepath FROM client_versions WHERE id = ?`, id)
	var path string
	if err := row.Scan(&path); err != nil {
		return "", err
	}
	_, err := db.Exec(`DELETE FROM client_versions WHERE id = ?`, id)
	return path, err
}

// --- Resource Packages ---

// ListResourcePackages 按创建时间降序返回所有资源包。
func ListResourcePackages(db *DB) ([]model.ResourcePackage, error) {
	rows, err := db.Query(`SELECT id, name, version, filename, filepath, is_latest, created_at FROM resource_packages ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.ResourcePackage, 0)
	for rows.Next() {
		var v model.ResourcePackage
		var isLatest int
		if err := rows.Scan(&v.ID, &v.Name, &v.Version, &v.Filename, &v.Filepath, &isLatest, &v.CreatedAt); err != nil {
			return nil, err
		}
		v.IsLatest = isLatest == 1
		list = append(list, v)
	}
	return list, rows.Err()
}

// CreateResourcePackage 插入一条新的资源包记录，并回填自增 ID。
func CreateResourcePackage(db *DB, v *model.ResourcePackage) error {
	res, err := db.Exec(
		`INSERT INTO resource_packages (name, version, filename, filepath, is_latest) VALUES (?, ?, ?, ?, ?)`,
		v.Name, v.Version, v.Filename, v.Filepath, boolToInt(v.IsLatest),
	)
	if err != nil {
		return err
	}
	v.ID, _ = res.LastInsertId()
	return nil
}

// DeleteResourcePackage 删除指定 id 的资源包记录，并返回其文件路径。
func DeleteResourcePackage(db *DB, id int64) (string, error) {
	row := db.QueryRow(`SELECT filepath FROM resource_packages WHERE id = ?`, id)
	var path string
	if err := row.Scan(&path); err != nil {
		return "", err
	}
	_, err := db.Exec(`DELETE FROM resource_packages WHERE id = ?`, id)
	return path, err
}

// SetLatestResourcePackage 清除所有资源包的最新标记，并将指定 id 设为最新。
func SetLatestResourcePackage(db *DB, id int64) error {
	_, err := db.Exec(`UPDATE resource_packages SET is_latest = 0`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE resource_packages SET is_latest = 1 WHERE id = ?`, id)
	return err
}

// GetLatestResourcePackage 返回标记为最新的资源包。
func GetLatestResourcePackage(db *DB) (*model.ResourcePackage, error) {
	row := db.QueryRow(`SELECT id, name, version, filename, filepath, is_latest, created_at FROM resource_packages WHERE is_latest = 1 LIMIT 1`)
	var v model.ResourcePackage
	var isLatest int
	err := row.Scan(&v.ID, &v.Name, &v.Version, &v.Filename, &v.Filepath, &isLatest, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	v.IsLatest = isLatest == 1
	return &v, nil
}

// UpdateResourcePackageName 更新指定资源包的显示名称。
func UpdateResourcePackageName(db *DB, id int64, name string) error {
	_, err := db.Exec(`UPDATE resource_packages SET name = ? WHERE id = ?`, name, id)
	return err
}

// --- Clients ---

// UpsertClient 根据客户端名称插入或更新客户端记录，存在时更新状态与版本等信息。
// 如果客户端从离线状态恢复在线，会自动重置 online_since 为当前时间。
func UpsertClient(db *DB, c *model.Client) error {
	row := db.QueryRow(`SELECT id, last_seen, online_since FROM clients WHERE name = ?`, c.Name)
	var id int64
	var lastSeen, onlineSince sql.NullTime
	err := row.Scan(&id, &lastSeen, &onlineSince)

	now := time.Now()
	onlineThreshold := 35 * time.Second

	if err == nil {
		// 已有记录：判断是否由离线恢复在线
		if lastSeen.Valid && now.Sub(lastSeen.Time) > onlineThreshold {
			onlineSince = sql.NullTime{Time: now, Valid: true}
		}
		c.ID = id
		c.OnlineSince = onlineSince.Time
		_, err := db.Exec(
			`UPDATE clients SET client_version=?, software_version=?, status=?, is_running=?, ip=?, os_version=?, memory=?, cpu=?, process_runtime=?, last_seen=?, online_since=?
			 WHERE id=?`,
			c.ClientVersion, c.SoftwareVersion, c.Status, boolToInt(c.IsRunning),
			c.IP, c.OSVersion, c.Memory, c.CPU, c.ProcessRuntime, now, onlineSince.Time, c.ID,
		)
		return err
	} else if err != sql.ErrNoRows {
		return err
	}

	// 新记录：上线时间设为当前
	c.OnlineSince = now
	res, err := db.Exec(
		`INSERT INTO clients (name, client_version, software_version, status, is_running, ip, os_version, memory, cpu, process_runtime, last_seen, online_since)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Name, c.ClientVersion, c.SoftwareVersion, c.Status, boolToInt(c.IsRunning),
		c.IP, c.OSVersion, c.Memory, c.CPU, c.ProcessRuntime, now, now,
	)
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

// GetClient 根据 id 查询单个客户端。
func GetClient(db *DB, id int64) (*model.Client, error) {
	row := db.QueryRow(`SELECT id, name, client_version, software_version, status, is_running, ip, os_version, memory, cpu, process_runtime, last_seen, online_since, created_at FROM clients WHERE id = ?`, id)
	c := &model.Client{}
	var isRunning int
	err := row.Scan(&c.ID, &c.Name, &c.ClientVersion, &c.SoftwareVersion, &c.Status, &isRunning,
		&c.IP, &c.OSVersion, &c.Memory, &c.CPU, &c.ProcessRuntime, &c.LastSeen, &c.OnlineSince, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.IsRunning = isRunning == 1
	return c, nil
}

// ListClients 按最近心跳时间降序返回所有客户端。
func ListClients(db *DB) ([]model.Client, error) {
	rows, err := db.Query(`SELECT id, name, client_version, software_version, status, is_running, ip, os_version, memory, cpu, process_runtime, last_seen, online_since, created_at FROM clients ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.Client, 0)
	for rows.Next() {
		var c model.Client
		var isRunning int
		if err := rows.Scan(&c.ID, &c.Name, &c.ClientVersion, &c.SoftwareVersion, &c.Status, &isRunning,
			&c.IP, &c.OSVersion, &c.Memory, &c.CPU, &c.ProcessRuntime, &c.LastSeen, &c.OnlineSince, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.IsRunning = isRunning == 1
		list = append(list, c)
	}
	return list, rows.Err()
}

// --- Commands ---

// CreateCommand 插入一条待执行的命令记录，并回填自增 ID。
func CreateCommand(db *DB, cmd *model.ClientCommand) error {
	res, err := db.Exec(
		`INSERT INTO client_commands (client_id, command_type, payload, status) VALUES (?, ?, ?, ?)`,
		cmd.ClientID, cmd.CommandType, cmd.Payload, cmd.Status,
	)
	if err != nil {
		return err
	}
	cmd.ID, _ = res.LastInsertId()
	return nil
}

// ListPendingCommands 返回指定客户端所有 pending 状态的命令，按创建时间升序排列。
func ListPendingCommands(db *DB, clientID int64) ([]model.ClientCommand, error) {
	rows, err := db.Query(
		`SELECT id, client_id, command_type, payload, status, created_at FROM client_commands WHERE client_id = ? AND status = 'pending' ORDER BY created_at ASC`,
		clientID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.ClientCommand, 0)
	for rows.Next() {
		var c model.ClientCommand
		if err := rows.Scan(&c.ID, &c.ClientID, &c.CommandType, &c.Payload, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// UpdateCommandStatus 更新指定命令的状态。
func UpdateCommandStatus(db *DB, id int64, status string) error {
	_, err := db.Exec(`UPDATE client_commands SET status = ? WHERE id = ?`, status, id)
	return err
}

// boolToInt 将布尔值转换为整数，true 为 1，false 为 0，用于 SQLite 存储。
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetClientByName 根据名称查询单个客户端。
func GetClientByName(db *DB, name string) (*model.Client, error) {
	row := db.QueryRow(`SELECT id, name, client_version, software_version, status, is_running, ip, os_version, memory, cpu, process_runtime, last_seen, online_since, created_at FROM clients WHERE name = ?`, name)
	c := &model.Client{}
	var isRunning int
	err := row.Scan(
		&c.ID, &c.Name, &c.ClientVersion, &c.SoftwareVersion, &c.Status, &isRunning,
		&c.IP, &c.OSVersion, &c.Memory, &c.CPU, &c.ProcessRuntime, &c.LastSeen, &c.OnlineSince, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.IsRunning = isRunning == 1
	return c, nil
}

// UpdateClientName 更新指定客户端的显示名称。
func UpdateClientName(db *DB, id int64, name string) error {
	_, err := db.Exec(`UPDATE clients SET name = ? WHERE id = ?`, name, id)
	return err
}

// DeleteClient 删除指定 id 的客户端记录。
func DeleteClient(db *DB, id int64) error {
	_, err := db.Exec(`DELETE FROM clients WHERE id = ?`, id)
	return err
}

// UpdateClientRunning 更新指定客户端的运行状态与进程运行时长。
func UpdateClientRunning(db *DB, id int64, running bool, runtime int64) error {
	_, err := db.Exec(`UPDATE clients SET is_running = ?, process_runtime = ? WHERE id = ?`, boolToInt(running), runtime, id)
	return err
}

// UpdateClientSoftwareVersion 更新指定客户端上被管理软件的版本号。
func UpdateClientSoftwareVersion(db *DB, id int64, version string) error {
	_, err := db.Exec(`UPDATE clients SET software_version = ? WHERE id = ?`, version, id)
	return err
}

// UpdateClientClientVersion 更新指定客户端程序自身的版本号。
func UpdateClientClientVersion(db *DB, id int64, version string) error {
	_, err := db.Exec(`UPDATE clients SET client_version = ? WHERE id = ?`, version, id)
	return err
}
