AddFunds
{
  "_id":"Txx1",
  "Id": "1",
  "CurrencyId": "3",
  "Account": [
    "Smartxx1"
  ],
  "DocType": "NIU",
  "Amount": 101,
  "Desc": "add funds"
}


RemoveFunds
{
  "_id":"Txx2",
  "Id": "1",
  "CurrencyId": "3",
  "Account": [
    "Smartxx1"
  ],
  "DocType": "NIU",
  "Amount": -10,
  "Desc": "add funds"
}

TransferFunds
{
    "_id":"Txx2",
    "DocType": "NIU",
    "Senders": ["Smartxx1"],
    "Receivers": ["Smartxx2"],
    "Id": "112",
    "InvokerId":"744FBCD4110980A0B7A68FB0B6D32CACB3F4A176",
    "Amount": 1
}