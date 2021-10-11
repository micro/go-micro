# Services

[Micro Services](https://github.com/micro/services) provide additional functionality via external third party services.

## Overview

Go Micro provides core abstractions for building distributed systems which are likely 
direct dependencies on locally hosted infrastructure. In the case of external services 
such as sending email, sms, etc we leave this up to the user. Yet they are becoming 
more core to all forms of distributed systems development beyond infrastructure.

This directory serves as guidance for accessing external Micro services managed 
and hosted by [m3o.com](https://m3o.com) and powered by [Micro](https://github.com/micro/micro).

## Features

The types of services available are as follows:

- [Address](https://m3o.com/address) - Address lookup by postcode
- [Answer](https://m3o.com/answer) - Instant answers to any question
- [Cache](https://m3o.com/cache) - Quick access key-value storage
- [Crypto](https://m3o.com/crypto) - Cryptocurrency prices, quotes, and news
- [Currency](https://m3o.com/currency) - Exchange rates and currency conversion
- [Db](https://m3o.com/db) - Simple database service
- [Email](https://m3o.com/email) - Send emails in a flash
- [Emoji](https://m3o.com/emoji) - All the emojis you need 🎉
- [File](https://m3o.com/file) - Store, list, and retrieve text files
- [Forex](https://m3o.com/forex) - Foreign exchange (FX) rates
- [Geocoding](https://m3o.com/geocoding) - Geocode an address to gps location and the reverse.
- [Helloworld](https://m3o.com/helloworld) - Just saying hello world
- [Holidays](https://m3o.com/holidays) - Find the holidays observed in a particular country
- [Id](https://m3o.com/id) - Generate unique IDs (uuid, snowflake, etc)
- [Image](https://m3o.com/image) - Quickly upload, resize, and convert images
- [Ip](https://m3o.com/ip) - IP to geolocation lookup
- [Location](https://m3o.com/location) - Real time GPS location tracking and search
- [Otp](https://m3o.com/otp) - One time password generation
- [Postcode](https://m3o.com/postcode) - Fast UK postcode lookup
- [Prayer](https://m3o.com/prayer) - Islamic prayer times
- [Qr](https://m3o.com/qr) - QR code generator
- [Quran](https://m3o.com/quran) - The Holy Quran
- [Routing](https://m3o.com/routing) - Etas, routes and turn by turn directions
- [Rss](https://m3o.com/rss) - RSS feed crawler and reader
- [Sentiment](https://m3o.com/sentiment) - Real time sentiment analysis
- [Sms](https://m3o.com/sms) - Send an SMS message
- [Stock](https://m3o.com/stock) - Live stock quotes and prices
- [Stream](https://m3o.com/stream) - Publish and subscribe to messages
- [Sunnah](https://m3o.com/sunnah) - Traditions and practices of the Islamic prophet, Muhammad (pbuh)
- [Thumbnail](https://m3o.com/thumbnail) - Create website thumbnails
- [Time](https://m3o.com/time) - Time, date, and timezone info
- [Twitter](https://m3o.com/twitter) - Realtime twitter timeline & search
- [Url](https://m3o.com/url) - URL shortening, sharing, and tracking
- [User](https://m3o.com/user) - User management and authentication
- [Weather](https://m3o.com/weather) - Real time weather forecast

## Explore

The source code for all services exist in [github.com/micro/services](https://github.com/micro/services).

The hosted versions of all services can be found on [m3o.com](https://m3o.com).

## Usage

- Head to [m3o.com](https://m3o.com) and signup for a free account. 
- Generate an API key on the [Settings page](https://m3o.com/settings/keys).
- Browse the APIs on the [Explore page](https://m3o.com/explore).
- Call any API using your token in the `Authorization: Bearer [Token]` header and `https://api.m3o.com/v1/[service]/[endpoint]` url.

## Example

Find the service you're looking for and browse to its API page e.g [https://m3o.com/currency/api](https://m3o.com/currency/api).

Copy the example for the relevant endpoint

```go
package main

import (
	"fmt"
	"os"

	"github.com/asim/go-micro/services/currency"
)

var (
	token = os.Getenv("MICRO_API_TOKEN")
)

func main() {
	// Convert returns the currency conversion rate between two pairs e.g USD/GBP
	currencyService := currency.NewCurrencyService(token)

	rsp, err := currencyService.Convert(&currency.ConvertRequest{
		From: "USD",
		To:   "GBP",
	})

	// {
	//	"from": "USD",
	//	"to": "GBP",
	//	"rate": 0.7104
	// }
	fmt.Println(rsp, err)
}
```
