package paste

const (
	sqlMigrate1 = `CREATE TABLE module_paste_data (id SERIAL PRIMARY KEY, content TEXT)`
	sqlMigrate2 = `CREATE TABLE module_paste_links (id SERIAL PRIMARY KEY, content TEXT)`

	// $1 = content
	// id sql.NullInt64
	sqlAddPaste = `INSERT INTO module_paste_data (content) VALUES ($1)
			RETURNING id`

	// $1 = id
	sqlGetPaste = `SELECT content FROM module_paste_data WHERE id = $1`

	// $1 = content
	// id sql.NullInt64
	sqlAddLink = `INSERT INTO module_paste_links (content) VALUES ($1)
			RETURNING id`

	// $1 = id
	sqlGetLink = `SELECT content FROM module_paste_links WHERE id = $1`
)

func (mod *PasteModule) GetPaste(id int64) (string, error) {
	var content string
	found := false
	mod.pasteLock.Lock()
	content, found = mod.pasteContent[id]
	mod.pasteLock.Unlock()
	if found {
		return content, nil
	}

	stmt, err := mod.team.DB().Prepare(sqlGetPaste)
	if err != nil {
		return "", err
	}
	defer stmt.Close()
	row := stmt.QueryRow(id)
	err = row.Scan(&content)
	if err != nil {
		return "", err
	}
	mod.pasteLock.Lock()
	mod.pasteContent[id] = content
	mod.pasteLock.Unlock()
	return content, nil
}

func (mod *PasteModule) CreatePaste(content string) (int64, error) {
	stmt, err := mod.team.DB().Prepare(sqlAddPaste)
	if err != nil {
		return -1, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(content)
	var id int64
	err = row.Scan(&id)
	if err != nil {
		return -1, err
	}

	mod.pasteLock.Lock()
	mod.pasteContent[id] = content
	mod.pasteLock.Unlock()
	return id, nil
}

func (mod *PasteModule) GetLink(id int64) (string, error) {
	var content string
	found := false
	mod.pasteLock.Lock()
	content, found = mod.linkContent[id]
	mod.pasteLock.Unlock()
	if found {
		return content, nil
	}

	stmt, err := mod.team.DB().Prepare(sqlGetLink)
	if err != nil {
		return "", err
	}
	defer stmt.Close()
	row := stmt.QueryRow(id)
	err = row.Scan(&content)
	if err != nil {
		return "", err
	}
	mod.pasteLock.Lock()
	mod.linkContent[id] = content
	mod.pasteLock.Unlock()
	return content, nil
}

func (mod *PasteModule) CreateLink(content string) (int64, error) {
	stmt, err := mod.team.DB().Prepare(sqlAddLink)
	if err != nil {
		return -1, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(content)
	var id int64
	err = row.Scan(&id)
	if err != nil {
		return -1, err
	}

	mod.pasteLock.Lock()
	mod.linkContent[id] = content
	mod.pasteLock.Unlock()
	return id, nil
}
