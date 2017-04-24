package main

import (
	"net/http"

	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"os"
	"strconv"
	"time"
)

var db *mgo.Database

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Transaction struct {
	Id        bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Sender    int           `json:"sender"`
	Receiver  int           `json:"receiver"`
	Timestamp int64         `json:"timestamp"`
	Sum       int           `json:"sum"`
}

type Routes []Route

var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		Index,
	},
	Route{
		"TransactionCreate",
		"POST",
		"/transactions/",
		TransactionsCreate,
	},
	Route{
		"TransactionList",
		"GET",
		"/transactions/",
		TransactionsList,
	},
	Route{
		"BalanceList",
		"GET",
		"/balance/",
		BalanceList,
	},
}

func main() {
	session, err := mgo.Dial("mongodb://database:27017/transacto")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	db = session.DB(os.Getenv("DATABASE_NAME"))

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	log.Fatal(http.ListenAndServe(":5000", router))
}

func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

func Index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/transactions/", http.StatusMovedPermanently)
}

func TransactionsCreate(w http.ResponseWriter, r *http.Request) {
	transactions := db.C("transactions")

	var transaction Transaction
	err := json.NewDecoder(r.Body).Decode(&transaction)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	/* Check for available post parameters */
	if transaction.Timestamp == 0 || transaction.Sum == 0 {
		http.Error(w, errors.New("Invalid post parameters").Error(), http.StatusUnprocessableEntity)
		return
	}

	/* Insert to collection */
	err = transactions.Insert(&transaction)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
	}

	/* Send back result */
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(transaction)
	if err != nil {
		http.Error(w, err.Error(), http.StatusCreated)
		log.Fatal(err)
	}
}

func TransactionsList(w http.ResponseWriter, r *http.Request) {
	transactions := db.C("transactions")

	params := r.URL.Query()

	/* Use user query param if it exists */
	query := bson.M{"_id": bson.M{"$exists": true}}
	if params.Get("user") != "" {
		if user, err := strconv.Atoi(params.Get("user")); err == nil {
			query["$or"] = []bson.M{
				{"sender": user},
				{"receiver": user},
			}

			/* Create index */
			index := mgo.Index{
				Key:        []string{"sender", "receiver"},
				Unique:     false,
				DropDups:   false,
				Background: true,
				Sparse:     true,
			}

			err := transactions.EnsureIndex(index)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	/* Use day query param if it exists */
	if params.Get("day") != "" {
		if day, err := strconv.ParseInt(params.Get("day"), 10, 64); err == nil {
			/* Convert timestamp to date and get start and end of its day */
			date := time.Unix(day, 0)
			start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
			end := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, time.UTC)

			query["timestamp"] = bson.M{"$gte": start.Unix(), "$lt": end.Unix()}
		}
	}

	/* Use threshold query param if it exists */
	if params.Get("threshold") != "" {
		if threshold, err := strconv.Atoi(params.Get("threshold")); err == nil {
			query["sum"] = bson.M{"$gte": threshold}
		}
	}

	/* Query */
	result := []Transaction{}
	err := transactions.Find(query).All(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}

	/* Send back response */
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusCreated)
		log.Fatal(err)
		return
	}
}

func BalanceList(w http.ResponseWriter, r *http.Request) {
	transactions := db.C("transactions")

	/* Use sparse compound index */
	index := mgo.Index{
		Key:        []string{"sender", "receiver"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	err := transactions.EnsureIndex(index)
	if err != nil {
		log.Fatal(err)
	}

	params := r.URL.Query()

	/* Make sure query params exist */
	user, err := strconv.Atoi(params.Get("user"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	since, err := strconv.ParseInt(params.Get("since"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	until, err := strconv.ParseInt(params.Get("until"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	/* Aggregate */
	pipe := transactions.Pipe([]bson.M{
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"sender": user},
					{"receiver": user},
				},
				"timestamp": bson.M{"$gte": since, "$lt": until},
			},
		},
		{
			"$group": bson.M{
				"_id": user,
				"balance": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$sender", user}},
							"$sum",
							bson.M{"$multiply": []interface{}{"$sum", -1}},
						},
					},
				},
			},
		},
	})

	result := []bson.M{}
	err = pipe.All(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}

	/* Send back response */
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusCreated)
		return
	}
}
