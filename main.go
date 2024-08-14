package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

type Domain struct {
	ID              int
	Domain, ExpDate string
}

func setUpDB() *sql.DB {
	db, err := sql.Open("mysql", strings.Join([]string{os.Getenv("DB_user"), ":", os.Getenv("DB_password"), "@tcp(", os.Getenv("DB_host"), ":3306)/domainmod"}, ""))
	if err != nil {
		log.Panic(err)
	}
	db.SetConnMaxLifetime(time.Minute * 4)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	return db
}

func getDomains(db *sql.DB) []Domain {
	rows, err := db.Query("SELECT domain, id, expiry_date FROM domains")
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	var domains []Domain
	for rows.Next() {
		var domain, expiry_date string
		var id int
		if err := rows.Scan(&domain, &id, &expiry_date); err != nil {
			log.Panic("failed to get domains from DB: ", err)
		}
		domains = append(domains, Domain{Domain: domain, ID: id, ExpDate: expiry_date})
	}
	if err := rows.Err(); err != nil {
		log.Panic(err)
	}
	return domains
}

func main() {
	for {
		db := setUpDB()
		domains := getDomains(db)

		currentDate := time.Now()
		for _, domain := range domains {
			ExpDate, err := time.Parse("2006-01-02", domain.ExpDate)
			if err != nil {
				log.Println(err)
				continue
			}
			diff := ExpDate.Sub(currentDate).Hours() / 24
			log.Println(domain.Domain)
			if int(diff) <= 25 {
				result, err := whois.Whois(domain.Domain)
				if err != nil {
					log.Println(err)
					continue
				}
				res, err := whoisparser.Parse(result)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Println(strings.Split(res.Domain.ExpirationDate, "T")[0], res.Registrar.Name, res.Domain.NameServers)
				var regID, accID int
				row := db.QueryRow("SELECT id FROM registrars WHERE name = ?", res.Registrar.Name)
				var id int
				err = row.Scan(&id)
				if err != nil {
					if err == sql.ErrNoRows {
						result, err := db.Exec("INSERT INTO registrars (name, url, notes, insert_time) VALUES (?, ?, 'Created by Go', ?)", res.Registrar.Name, res.Registrar.ReferralURL, currentDate.Format("2006-01-02"))
						if err != nil {
							log.Println(err)
							continue
						}
						insertedID, err := result.LastInsertId()
						if err != nil {
							log.Println(err)
							continue
						}
						regID = int(insertedID)
					} else {
						log.Println(err)
						continue
					}
				} else {
					regID = id
				}
				row = db.QueryRow("SELECT id FROM registrar_accounts WHERE registrar_id = ?", id)
				err = row.Scan(&id)
				if err != nil {
					if err == sql.ErrNoRows {
						result, err := db.Exec("INSERT INTO registrar_accounts (owner_id, registrar_id, username, email_address, password, account_id, reseller_id, api_app_name, api_key, api_secret, notes, insert_time) VALUES (1, ?, 'Default', 'none@none.com', 'Default', '', '', '', '', '', 'Created by Go', ?)", id, currentDate.Format("2006-01-02"))
						if err != nil {
							log.Println(err)
							continue
						}
						insertedID, err := result.LastInsertId()
						if err != nil {
							log.Println(err)
							continue
						}
						accID = int(insertedID)
					} else {
						log.Println(err)
						continue
					}
				} else {
					accID = id
				}
				r, err := db.Exec("UPDATE domains SET expiry_date = ?, registrar_id = ?, update_time = ?, account_id = ? WHERE id = ?", strings.Split(res.Domain.ExpirationDate, "T")[0], regID, currentDate.Format("2006-01-02"), accID, domain.ID)
				if err != nil {
					log.Println(err)
					continue
				}
				r1, _ := r.RowsAffected()
				log.Println("Updated", r1)
				time.Sleep(20 * time.Second)
			}
		}
		db.Close()
		log.Println("Done. Waiting for next run... \nSleeping for 220 hours")
		// sleep ~10 days
		time.Sleep(220 * time.Hour)
	}
}
