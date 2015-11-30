package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
)

var sqlString string = "gimvic:GimVicServer@/gimvic"

func main() {
	http.HandleFunc("/chooserOptions", chooserOptions)
	http.ListenAndServe(":8080", nil)
}

func chooserOptions(w http.ResponseWriter, r *http.Request) {
	response := ChooserOptionsResponse{}
	con, err := sql.Open("mysql", sqlString)
	check(err)
	defer con.Close()

	//fill main classes
	rows, err := con.Query("select class from classes where main=1;")
	check(err)
	for rows.Next() {
		var temp string
		rows.Scan(&temp)
		response.MainClasses = append(response.MainClasses, temp)
	}

	//fill additional classes
	rows, err = con.Query("select class from classes where main=0;")
	check(err)
	for rows.Next() {
		var temp string
		rows.Scan(&temp)
		response.AdditionalClasses = append(response.AdditionalClasses, temp)
	}

	responseStr, err := json.Marshal(response)
	check(err)
	fmt.Fprint(w, string(responseStr))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type ChooserOptionsResponse struct {
	MainClasses       []string
	AdditionalClasses []string
}
