package main

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "domainmod:{PASSWORD}@db/domainmod")
	if err != nil {
		panic(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	defer db.Close()
	//  get domains colum from a db table calls domains and put it in a array
	rows, err := db.Query("SELECT domain FROM domains")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			panic(err)
		}
		domains = append(domains, domain)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	// print the array
	for _, domain := range domains {
		println(domain)
	}
}
