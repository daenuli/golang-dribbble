
package main

import (
	"log"
	"net/http"
	"text/template"
	"io/ioutil"
	"fmt"
	"bytes"
    "encoding/json"
	"os"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Dribbble struct {
	Access string `json:"access_token"`
	Id int64 `json:"id"`
	Avatar string `json:"avatar_url"`
	Name string `json:"name"`
	Url string `json:"html_url"`
	Location string `json:"location"`
	Username string `json:"login"`
}

type Users struct {
	Id int `json:"id"`
	Avatar string `json:"avatar_url"`
	Name string `json:"name"`
	Url string `json:"html_url"`
	Location string `json:"location"`
	Username string `json:"login"`
}

var tmpl= template.Must(template.ParseGlob("views/*"))
var clientid string = "46662781a807238a23a3060b87ecc557775e4a399051314e16c1f02cb8576a58"
var clientsecret string = "36f3f4586b6a1a19bace5c1c71e86bccc6cb697c530e5f7d14fc44b22aeaff3d"
// var redirecturl string = "http://127.0.0.1:8000"
var url string = "http://127.0.0.1:8000"
var api string = "https://api.dribbble.com/v2/"
var scope string = "public+upload"

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
    w.WriteHeader(status)
    if status == http.StatusNotFound {
        fmt.Fprint(w, "page not found")
    }
}

func Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
        errorHandler(w, r, http.StatusNotFound)
        return
    }

	Data := map[string]interface{}{
		"client_id": clientid,
		"redirect": url+"/callback",
		"scope": scope,
	}
	tmpl.ExecuteTemplate(w, "Index", Data)
}
func Callback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/callback" {
        errorHandler(w, r, http.StatusNotFound)
        return
    }

	codes, ok := r.URL.Query()["code"]
	if !ok || len(codes[0]) < 1 {
        log.Println("Url Param 'code' is missing")
        return
    }

    code := codes[0]

	jsonData := map[string]string{"client_id": clientid, "client_secret": clientsecret, "code": code}
    jsonValue, _ := json.Marshal(jsonData)
    response, err := http.Post("https://dribbble.com/oauth/token", "application/json", bytes.NewBuffer(jsonValue))
    if err != nil {
        fmt.Printf("The HTTP request failed with error %s\n", err)
    } else {
		defer response.Body.Close()
		body_byte, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}

		body := Dribbble{}
		errs := json.Unmarshal(body_byte, &body)
		if errs != nil {
			panic(errs)
		}

		var at = "/home?access_token="+body.Access

		http.Redirect(w, r, url+at, http.StatusSeeOther)
    }
}


func Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/home" {
        errorHandler(w, r, http.StatusNotFound)
        return
    }

	access, ok := r.URL.Query()["access_token"]
	if !ok || len(access[0]) < 1 {
        log.Println("Url Param 'access_token' is missing")
        return
    }

    token := access[0]


	Data := map[string]interface{}{
		"access_token": token,
		// "url": url,
		// "user_api": api+"user?access_token="+token,
		// "user_url": url+"/user?access_token="+token,
	}
	tmpl.ExecuteTemplate(w, "Home", Data)

}

func User(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/user" {
        errorHandler(w, r, http.StatusNotFound)
        return
    }

    db, err := sql.Open("sqlite3", "dribbble.db")
    rows, err := db.Query("SELECT * FROM user")

    if err != nil {
        panic(err.Error())
    }
    usr := Users{}
    res := []Users{}
    for rows.Next() {
        var id int
        var avatar_url, name, html_url, location, login string
        err = rows.Scan(&id, &avatar_url, &name, &html_url, &location, &login)
        if err != nil {
            panic(err.Error())
        }
        usr.Id = id
        usr.Avatar = avatar_url
        usr.Name = name
        usr.Url = html_url
        usr.Location = location
        usr.Username = login
        res = append(res, usr)
    }
    tmpl.ExecuteTemplate(w, "User", res)
}

func Save(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/save" {
        errorHandler(w, r, http.StatusNotFound)
        return
    }

	access, ok := r.URL.Query()["access_token"]
	if !ok || len(access[0]) < 1 {
        log.Println("Url Param 'access_token' is missing")
        return
    }

    token := access[0]

	response, err := http.Get(api+"user?access_token="+token)
    if err != nil {
        fmt.Printf("The HTTP request failed with error %s\n", err)
    } else {
		defer response.Body.Close()
		body_byte, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}

		body := Dribbble{}
		errs := json.Unmarshal(body_byte, &body)
		if errs != nil {
			panic(errs)
		}

		db, err := sql.Open("sqlite3", "dribbble.db")
		if err != nil {
			log.Fatal(err)
		}
		
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("insert into user(avatar_url, name, html_url, location, login) values(?, ?, ?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		_, errs = stmt.Exec(body.Avatar, body.Name, body.Url, body.Location, body.Username)
		if errs != nil {
			log.Fatal(errs)
		}
		tx.Commit()
		var at = "/user?access_token="+token

		db.Close()
		http.Redirect(w, r, url+at, http.StatusSeeOther)
	}
}

func Delete(w http.ResponseWriter, r *http.Request) {
	access, ok := r.URL.Query()["access_token"]
	if !ok || len(access[0]) < 1 {
        log.Println("Url Param 'access_token' is missing")
        return
    }

    token := access[0]

	id := r.FormValue("id")
    db, err := sql.Open("sqlite3", "dribbble.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("delete from user where id=?", id)

    if err != nil {
		log.Fatal(err)
    }
	var at = "/user?access_token="+token
	http.Redirect(w, r, url+at, http.StatusSeeOther)
}

func Getpopular(w http.ResponseWriter, r *http.Request) {
	access, ok := r.URL.Query()["access_token"]
	if !ok || len(access[0]) < 1 {
        log.Println("Url Param 'access_token' is missing")
        return
    }

    token := access[0]
	var at = "popular_shots?access_token="+token
	http.Redirect(w, r, api+at, http.StatusSeeOther)
}

func Search(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/search" {
	    errorHandler(w, r, http.StatusNotFound)
	    return
	}

	name := r.FormValue("name")
 

	db, err := sql.Open("sqlite3", "dribbble.db")
    rows, err := db.Query("SELECT * FROM user where name like '%' || $1 || '%'", name)

    if err != nil {
        panic(err.Error())
    }
    usr := Users{}
    res := []Users{}
    for rows.Next() {
        var id int
        var avatar_url, name, html_url, location, login string
        err = rows.Scan(&id, &avatar_url, &name, &html_url, &location, &login)
        if err != nil {
            panic(err.Error())
        }
        usr.Id = id
        usr.Avatar = avatar_url
        usr.Name = name
        usr.Url = html_url
        usr.Location = location
        usr.Username = login
        res = append(res, usr)
    }
    tmpl.ExecuteTemplate(w, "Search", res)
}


func Popular(w http.ResponseWriter, r *http.Request) {
if r.URL.Path != "/popular" {
        errorHandler(w, r, http.StatusNotFound)
        return
    }
    tmpl.ExecuteTemplate(w, "Popular", nil)
}

func listuser(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/listuser" {
        errorHandler(w, r, http.StatusNotFound)
        return
    }
}

func main() {
	os.Remove("dribbble.db")

	db, err := sql.Open("sqlite3", "dribbble.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
	create table user (id integer not null primary key autoincrement, avatar_url text, name text, html_url text, location text, login text);
	delete from user;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	log.Println("Server started on: http://127.0.0.1:8000")
	http.HandleFunc("/", Index)
	http.HandleFunc("/callback", Callback)
	http.HandleFunc("/home", Home)
	http.HandleFunc("/user", User)
	http.HandleFunc("/popular", Popular)
	http.HandleFunc("/search", Search)
	http.HandleFunc("/save", Save)
	http.HandleFunc("/getpopular", Getpopular)
	http.HandleFunc("/delete", Delete)
	// http.HandleFunc("/popular", Popular)
	log.Fatal(http.ListenAndServe(":8000", nil))
}