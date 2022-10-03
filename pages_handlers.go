package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	resp "github.com/vano2903/ipaas/responser"
)

// homepage handler
func (h Handler) HomePageHandler(w http.ResponseWriter, r *http.Request) {
	//return the home html page
	http.ServeFile(w, r, "./pages/home.html")
}

func (h Handler) LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	//return the login html page
	http.ServeFile(w, r, "./pages/login.html")
}

func (h Handler) UserPageHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectToDB()
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Client().Disconnect(context.TODO())

	//get the access token from the cookie
	cookie, err := r.Cookie("ipaas-access-token")
	if err != nil {
		if err == http.ErrNoCookie {
			//new token pair
			cookie, err := r.Cookie("ipaas-refresh-token")
			if err != nil {
				if err == http.ErrNoCookie {
					//resp.Error(w, http.StatusBadRequest, "No refresh token")
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				resp.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			refreshToken := cookie.Value

			//check if there is a refresh token
			if refreshToken == "" {
				resp.Error(w, 498, "No refresh token, do login again")
				return
			}

			//check if the refresh token is expired
			isExpired, err := IsRefreshTokenExpired(refreshToken, db)
			if err != nil {
				resp.Errorf(w, http.StatusInternalServerError, "Error checking if refresh token is expired: %s", err.Error())
			}
			if isExpired {
				//!should redirect to the oauth page
				resp.Error(w, 498, "Refresh token is expired")
				return
			}

			//generate a new token pair
			accessToken, newRefreshToken, err := GenerateNewTokenPairFromRefreshToken(refreshToken, db)
			if err != nil {
				resp.Error(w, http.StatusInternalServerError, err.Error())
				return
			}

			//delete the old tokens from the cookies
			http.SetCookie(w, &http.Cookie{
				Name:    "ipaas-access-token",
				Path:    "/",
				Value:   "",
				Expires: time.Unix(0, 0),
			})
			http.SetCookie(w, &http.Cookie{
				Name:    "ipaas-refresh-token",
				Path:    "/",
				Value:   "",
				Expires: time.Unix(0, 0),
			})

			//set the new tokens
			//!should set domain and path
			http.SetCookie(w, &http.Cookie{
				Name:    "ipaas-access-token",
				Path:    "/",
				Value:   accessToken,
				Expires: time.Now().Add(time.Hour),
			})
			http.SetCookie(w, &http.Cookie{
				Name:    "ipaas-refresh-token",
				Path:    "/",
				Value:   newRefreshToken,
				Expires: time.Now().Add(time.Hour * 24 * 7),
			})
			//make the user refresh the page
			http.Redirect(w, r, "/user/", http.StatusFound)
			return
		}
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	accessToken := cookie.Value

	//get the student generic infos from the access token
	student, err := GetUserFromAccessToken(accessToken, db)
	fmt.Println("studente:", student)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	//load page with the student info
	//template
	t, err := template.ParseFiles("./pages/user.html")
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// var rows *sql.Rows
	// rows, err = db.Query("SELECT containerID, type, name, description, isPublic FROM applications WHERE studentID = ?", student.ID)
	// if err != nil {
	// 	resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
	// 	return
	// }

	// //parse the rows into a []Application and return it
	// var applications []Application
	// var databases []Application
	// for rows.Next() {
	// 	var app Application
	// 	err = rows.Scan(&app.ContainerID, &app.Type, &app.Name, &app.Description, &app.IsPublic)
	// 	if err != nil {
	// 		resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
	// 		return
	// 	}
	// 	if app.Type == "database" {
	// 		databases = append(databases, app)
	// 	} else {
	// 		applications = append(applications, app)
	// 	}
	// }

	// var appHTMLString string
	// var dbHTMlstring string

	// //parse the applications into html
	// for _, app := range applications {
	// 	var btn string
	// 	if app.IsPublic {
	// 		btn = `<button type="button" onclick="makePrivate('%s') class="btn btn-warning">Make private</button>`
	// 	} else {
	// 		btn = `<button type="button" onclick="makePublic('%s') class="btn btn-info">Make public</button>`
	// 	}
	// 	appHTMLString += fmt.Sprintf(`
	// 	<div id="%s" class=\"doc\">
	// 		<p>%s</p>
	// 		%s
	// 		<button type="button" onclick="deleteContainer('%s')" class="btn btn-danger">Delete</button>
	// 		<hr>
	// 		<h5>%s</h5>
	// 	</div`, app.ContainerID, app.Name, btn, app.ContainerID, app.ContainerID, app.Description)
	// }

	// for _, database := range databases {
	// 	dbHTMlstring += fmt.Sprintf(`
	// 	<div id="%s" class=\"doc\">
	// 		<p>%s</p>
	// 		<button type="button" class="btn btn-info" disabled>Esporta</button>
	// 		<button type="button" onclick="deleteContainer('%s')" class="btn btn-danger">Delete</button>
	// 	</div`, database.ContainerID, database.Name, database.ContainerID)
	// }

	toParse := struct {
		Name string
		Pfp  string
		// Apps string
		// DBs  string
	}{
		Name: student.Name + " " + student.LastName,
		Pfp:  student.Pfp,
		// Apps: appHTMLString,
		// DBs:  dbHTMlstring,
	}

	t.Execute(w, toParse)
}

func (h Handler) NewAppPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./pages/newApp.html")
}

func (h Handler) NewDatabasePageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./pages/newDB.html")
}

func (h Handler) PublicStudentPageHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["studentID"]

	db, err := connectToDB()
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	//get the student generic infos from the access token
	idI, err := strconv.Atoi(id)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid student id")
		return
	}
	student, err := GetStudentFromID(idI, db)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	//load page with the student info
	//template
	t, err := template.ParseFiles("./pages/student.html")
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	toParse := struct {
		StudentID string
		Name      string
		LastName  string
	}{
		StudentID: id,
		Name:      student.Name,
		LastName:  student.LastName,
	}

	fmt.Println(toParse)

	t.Execute(w, toParse)
}
