package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var adminsCollection *mongo.Collection
var studentsCollection *mongo.Collection

type Admin struct {
	Username string
	Password string
	Email    string
	Token    string
}

func main() {

	//connect to MONGODB
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB!")

	adminsCollection = client.Database("testino").Collection("admins")
	studentsCollection = client.Database("testino").Collection("students")

	r := mux.NewRouter()
	r.HandleFunc("/ping", PingHandler).Methods("GET")
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/register", RegisterHandler).Methods("POST")

	http.Handle("/", r)
	fmt.Println("listening on port 5000")
	http.ListenAndServe(":5000", nil)
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "pong"})

	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> pong")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {

	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	userType := r.FormValue("userType")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if userType == "admin" {

		filter := bson.D{{Key: "username", Value: username},
			{Key: "password", Value: password}}
		var result Admin

		err := adminsCollection.FindOne(context.TODO(), filter).Decode(&result)
		if err != nil {
			fmt.Print("Invalid login data: ", username)
			http.Error(w, "نام کاربری یا رمز عبور اشتباه است.", http.StatusBadRequest)

		} else {
			json.NewEncoder(w).Encode(map[string]string{"token": result.Token})
			fmt.Print("logged in: ", username)
		}

	} else {

	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	filter := bson.D{{Key: "email", Value: email}}
	var result Admin

	err := adminsCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err == nil {
		http.Error(w, "این پست الکترونیک قبل استفاده شده است.", http.StatusBadRequest)
		fmt.Print("used email: " + email)

	} else {

		var result Admin
		filter := bson.D{{Key: "username", Value: username}}
		err := adminsCollection.FindOne(context.TODO(), filter).Decode(&result)
		if err == nil {
			http.Error(w, "این نام کاربری قبل استفاده شده است.", http.StatusBadRequest)
			fmt.Print("used username: " + username)

		} else {
			//REGISTER

			//generat token
			type customClaims struct {
				Username string `json:username`
				jwt.StandardClaims
			}
			claims := customClaims{
				Username: username,
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: 15000,
					Issuer:    "testino",
				},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

			signedToken, _ := token.SignedString([]byte("testino"))

			insertUser := Admin{username, password, email, signedToken}

			insertResult, err := adminsCollection.InsertOne(context.TODO(), insertUser)
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			if insertResult != nil {
				fmt.Print("New user added: ", email)
				json.NewEncoder(w).Encode(map[string]string{"token": signedToken})
			}
		}

	}

}
