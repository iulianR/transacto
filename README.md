### Go version - found in app directory

For each method of each route a handler is associated.

The TransactionCreate handler, which treats a POST message on /transactions/ is pretty straightforward decoding the POST body and inserting transaction data directly into the database. The timestamp is saved as an int64.

The TransactionList handler treats a GET message on /transactions/, taking into account the possible query parameters. GET queries also work when only a part of the query parameters are provided (e.g. get all users with id 1, timestamp and sum are ignored). The server assumes that the timestamp query parameter is sent as an integer representing a UNIX time value for that day. Some processing is done on it to obtain the day the instant happened before querying the database.

The BalanceList handler treats a GET message on /balance/. For simplicity, it assumes that all query parameters exist in the URL. An aggregation is performed to obtain the balance of an user for a given timeframe. The aggregation computes a sum of all transactions that the user partook in, making sure to multiply with -1 those transactions in which the user executed a payment. There is no processing done in memory.

For bonus, I used indexes to speed up queries. For BalanceList, I used a Sparse Index on sum to ignore transac

### Extra: Python version - found in python-version directory

My go-to language is Python when it comes to web development, so at first I started implementing the homework in Python/Flask. I realized towards the end that this would be a cool mini project to practice my Golang skills on so I switched over. The python version is mostly complete, only missing Indexes and proper error checking.

#### Installation
    python3.5 -m venv env
    . env/bin/activate
    pip install -r requirements.txt
    python3.5 api.py
