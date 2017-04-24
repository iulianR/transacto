from datetime import datetime

from flask import Flask, redirect, url_for
from flask_pymongo import PyMongo
from flask_restful import Api, Resource, reqparse, fields, marshal_with


app = Flask(__name__)
app.config['MONGO_DBNAME'] = "transacto"
mongo = PyMongo(app, config_prefix='MONGO')
APP_URL = "http://127.0.0.1:5000"


parser = reqparse.RequestParser()
parser.add_argument('user', type=int, required=True)
parser.add_argument('day', type=int, required=True)
parser.add_argument('threshold', type=int, required=True)

post_parser = reqparse.RequestParser()
post_parser.add_argument('sender', type=int, required=True, location='form')
post_parser.add_argument('receiver', type=int, required=True, location='form')
post_parser.add_argument('timestamp', type=int, required=True, location='form')
post_parser.add_argument('sum', type=int, required=True, location='form')

balance_parser = reqparse.RequestParser()
balance_parser.add_argument('user', type=int, required=True)
balance_parser.add_argument('since', type=int, required=True)
balance_parser.add_argument('until', type=int, required=True)

transaction_fields = {
    'sender': fields.Integer,
    'receiver': fields.Integer,
    'timestamp': fields.DateTime,
    'sum': fields.Integer
}

balance_fields = {
    'total': fields.Integer
}


class Transaction(Resource):
    @marshal_with(transaction_fields)
    def get(self):
        args = parser.parse_args()

        user_id = args['user']
        threshold = args['threshold']
        date = datetime.fromtimestamp(args['day'])
        start = date.replace(hour=0, minute=0, second=0)
        end = date.replace(hour=23, minute=59, second=59)

        cursor = mongo.db.transaction.find({
            '$or': [
                    {'sender': user_id},
                    {'receiver': user_id}
                ],
            'sum': {'$gt': threshold},
            'timestamp': {'$gte': start, '$lt': end}
        })
        data = []
        for transaction in cursor:
            transaction['_id'] = str(transaction['_id'])
            data.append(transaction)

        return data, 200

    def post(self):
        args = post_parser.parse_args(strict=True)

        mongo.db.transaction.insert({
            'sender': args['sender'],
            'receiver': args['receiver'],
            'timestamp': datetime.fromtimestamp(args['timestamp']),
            'sum': args['sum']
        })

        return {"response": "ok"}


class Balance(Resource):
    @marshal_with(balance_fields)
    def get(self):
        args = balance_parser.parse_args(strict=True)

        start = datetime.fromtimestamp(args['since'])
        end = datetime.fromtimestamp(args['until'])
        cursor = mongo.db.transaction.aggregate([
            {
                '$match': {
                    '$or': [
                        {'sender': args['user']},
                        {'receiver': args['user']}
                    ],
                    'timestamp': {'$gte': start, '$lt': end},
                }
            },
            {
                '$group': {
                    '_id': args['user'],
                    'total': {
                        '$sum': {
                            '$cond': [
                                {'$eq': ['$sender', args['user']]},
                                '$sum',
                                {'$multiply': ['$sum', -1]}
                            ],
                        }
                    }
                }
            }
        ])

        data = []
        for t in cursor:
            data.append(t)

        return data


class Index(Resource):
    def get(self):
        return redirect(url_for('transactions'))


api = Api(app)
api.add_resource(Index, '/', endpoint='index')
api.add_resource(Transaction, '/transactions/', endpoint='transactions')
api.add_resource(Balance, '/balance/', endpoint='balance')

if __name__ == '__main__':
    app.run(debug=True)
