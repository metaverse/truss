package mysql

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	MAX_MYSQL_RETRY_ATTEMPTS = 5
	MYSQL_DATETIME_FORMAT    = "2006-01-02 15:04:05"
	CACHE_RES_TTL_FOUND      = 3600
	CACHE_RES_TTL_MISS       = 600
	DEFAULT_PORT             = 3306
)

type MapRow func(*sql.Rows) (interface{}, error)
type JsonDecode func([]byte) ([]interface{}, error)

type Client struct {
	Host   string
	Port   int
	User   string
	Pass   string
	DbName string
	db     *sql.DB
}

func (c *Client) Close() {
	if c != nil && c.db != nil {
		// TODO: Requires a better workaround, mysql connections left hanging, but issues with queries in certain conditions
		c.db.Close()
		c.db = nil
	}
}

func (c *Client) GetUrl() string {
	return c.User + ":" + c.Pass + "@tcp(" + c.Host + ":" + strconv.Itoa(c.Port) + ")/"
}

func (c *Client) GetConn() (*sql.DB, error) {
	var err error
	if c.db == nil {
		c.db, err = sql.Open("mysql", fmt.Sprintf("%s%s?charset=utf8&parseTime=True", c.GetUrl(), c.DbName))
	}
	return c.db, err
}

func NewClient() (*Client, error) {
	host := "127.0.0.1"
	if eval := os.Getenv("MYSQL_HOST"); eval != "" {
		host = eval
	}

	port := 3306
	if eval := os.Getenv("MYSQL_PORT"); eval != "" {
		port, _ = strconv.Atoi(eval)
	}

	user := ""
	if eval := os.Getenv("MYSQL_USER"); eval != "" {
		user = eval
	}

	pass := ""
	if eval := os.Getenv("MYSQL_PASS"); eval != "" {
		pass = eval
	}

	dbname := ""
	if eval := os.Getenv("MYSQL_DB"); eval != "" {
		dbname = eval
	}

	return &Client{
		Host:   host,
		Port:   port,
		User:   user,
		Pass:   pass,
		DbName: dbname,
	}, nil
}

func QueryRows(c *Client, qStr string, args []interface{}, mfn MapRow) ([]interface{}, error) {

	results := []interface{}{}

	db, err := c.GetConn()
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare(qStr)
	if err != nil {
		return results, fmt.Errorf("Query prepare failed %s - %v", qStr, err)
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return results, fmt.Errorf("Query failed %s - %v", qStr, err)
	}
	defer rows.Close()

	for rows.Next() {
		entry, err := mfn(rows)
		if err != nil && strings.Contains(err.Error(), "no rows") == false {
			return results, fmt.Errorf("Query scan failed %s - %v", qStr, err)
		}

		results = append(results, entry)
	}

	if err = rows.Err(); err != nil && strings.Contains(err.Error(), "no rows") == false {
		return results, fmt.Errorf("Query rows failed %s - %v", qStr, err)
	}

	return results, nil
}

func Insert(c *Client, table string, params map[string]interface{}, tx *sql.Tx) (int64, error) {
	fs := []string{}
	vps := []string{}
	args := []interface{}{}
	for f, v := range params {
		fs = append(fs, f)
		vps = append(vps, "?")
		args = append(args, v)
	}

	qStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", table, strings.Join(fs, ","), strings.Join(vps, ","))

	var err error
	var res sql.Result
	if tx != nil {
		res, err = tx.Exec(qStr, args...)
	} else {
		db, derr := c.GetConn()
		if err != nil {
			err = derr
		} else {
			res, err = db.Exec(qStr, args...)
		}
	}

	var id int64
	if err != nil {
		return 0, fmt.Errorf("Query failed %s - %v", qStr, err)
	} else {
		id, err = res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("LastInsertId failed %s - %v", qStr, err)
		}
	}

	return id, nil
}

func Update(c *Client, table string, params map[string]interface{}, whereStr string, whereArgs []interface{}, tx *sql.Tx) error {
	fs := []string{}
	args := []interface{}{}
	for f, v := range params {
		fs = append(fs, f+" = ?")
		args = append(args, v)
	}

	for _, a := range whereArgs {
		args = append(args, a)
	}

	qStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, strings.Join(fs, ","), whereStr)

	var err error
	if tx != nil {
		_, err = tx.Exec(qStr, args...)
	} else {
		db, derr := c.GetConn()
		if err != nil {
			err = derr
		} else {
			_, err = db.Exec(qStr, args...)
		}
	}

	if err != nil {
		return fmt.Errorf("Query failed %s - %v", qStr, err)
	}

	return nil
}

func Exec(c *Client, qStr string, args []interface{}, tx *sql.Tx) error {

	var err error
	if tx != nil {
		_, err = tx.Exec(qStr, args...)
	} else {
		db, derr := c.GetConn()
		if err != nil {
			err = derr
		} else {
			_, err = db.Exec(qStr, args...)
		}
	}

	return err
}
