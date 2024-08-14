package main

import (
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

type Domain struct {
	ID      int
	Domain  string
	ExpDate string
}

func setUpDB() *sql.DB {
	db, err := sql.Open("mysql", "domainmod:vVZraEnJsuYz37J5Ge0K@tcp(db:3306)/domainmod")
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
				var regID int
				rows, err := db.Query("SELECT id FROM registrars WHERE name = ?", res.Registrar.Name)
				if err != nil && err != sql.ErrNoRows {
					log.Println(err)
					continue
				} else if err == sql.ErrNoRows {
					row, err := db.Exec("INSERT INTO registrars (name, url, notes) VALUES (?, ?, 'None')", res.Registrar.Name, res.Registrar.ReferralURL)
					if err != nil {
						log.Println(err)
						continue
					}
					id, err := row.LastInsertId()
					if err != nil {
						log.Println(err)
						continue
					}
					regID = int(id)
				}
				rows.Scan(&regID)
				r, err := db.Exec("UPDATE domains SET expiry_date = ?, registrar_id = ?, update_time = ? WHERE id = ?", strings.Split(res.Domain.ExpirationDate, "T")[0], regID, time.Now().Format("2006-01-02"), domain.ID)
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
		// sleep ~10 days
		time.Sleep(220 * time.Hour)
	}
}
