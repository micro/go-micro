package services

import (
	"github.com/asim/go-micro/v3/services/address"
	"github.com/asim/go-micro/v3/services/answer"
	"github.com/asim/go-micro/v3/services/cache"
	"github.com/asim/go-micro/v3/services/crypto"
	"github.com/asim/go-micro/v3/services/currency"
	"github.com/asim/go-micro/v3/services/db"
	"github.com/asim/go-micro/v3/services/email"
	"github.com/asim/go-micro/v3/services/emoji"
	"github.com/asim/go-micro/v3/services/evchargers"
	"github.com/asim/go-micro/v3/services/file"
	"github.com/asim/go-micro/v3/services/forex"
	"github.com/asim/go-micro/v3/services/geocoding"
	"github.com/asim/go-micro/v3/services/helloworld"
	"github.com/asim/go-micro/v3/services/holidays"
	"github.com/asim/go-micro/v3/services/id"
	"github.com/asim/go-micro/v3/services/image"
	"github.com/asim/go-micro/v3/services/ip"
	"github.com/asim/go-micro/v3/services/location"
	"github.com/asim/go-micro/v3/services/notes"
	"github.com/asim/go-micro/v3/services/otp"
	"github.com/asim/go-micro/v3/services/postcode"
	"github.com/asim/go-micro/v3/services/prayer"
	"github.com/asim/go-micro/v3/services/qr"
	"github.com/asim/go-micro/v3/services/quran"
	"github.com/asim/go-micro/v3/services/routing"
	"github.com/asim/go-micro/v3/services/rss"
	"github.com/asim/go-micro/v3/services/sentiment"
	"github.com/asim/go-micro/v3/services/sms"
	"github.com/asim/go-micro/v3/services/stock"
	"github.com/asim/go-micro/v3/services/stream"
	"github.com/asim/go-micro/v3/services/sunnah"
	"github.com/asim/go-micro/v3/services/thumbnail"
	"github.com/asim/go-micro/v3/services/time"
	"github.com/asim/go-micro/v3/services/twitter"
	"github.com/asim/go-micro/v3/services/url"
	"github.com/asim/go-micro/v3/services/user"
	"github.com/asim/go-micro/v3/services/vehicle"
	"github.com/asim/go-micro/v3/services/weather"
)

func NewClient(token string) *Client {
	return &Client{
		token: token,

		AddressService:    address.NewAddressService(token),
		AnswerService:     answer.NewAnswerService(token),
		CacheService:      cache.NewCacheService(token),
		CryptoService:     crypto.NewCryptoService(token),
		CurrencyService:   currency.NewCurrencyService(token),
		DbService:         db.NewDbService(token),
		EmailService:      email.NewEmailService(token),
		EmojiService:      emoji.NewEmojiService(token),
		EvchargersService: evchargers.NewEvchargersService(token),
		FileService:       file.NewFileService(token),
		ForexService:      forex.NewForexService(token),
		GeocodingService:  geocoding.NewGeocodingService(token),
		HelloworldService: helloworld.NewHelloworldService(token),
		HolidaysService:   holidays.NewHolidaysService(token),
		IdService:         id.NewIdService(token),
		ImageService:      image.NewImageService(token),
		IpService:         ip.NewIpService(token),
		LocationService:   location.NewLocationService(token),
		NotesService:      notes.NewNotesService(token),
		OtpService:        otp.NewOtpService(token),
		PostcodeService:   postcode.NewPostcodeService(token),
		PrayerService:     prayer.NewPrayerService(token),
		QrService:         qr.NewQrService(token),
		QuranService:      quran.NewQuranService(token),
		RoutingService:    routing.NewRoutingService(token),
		RssService:        rss.NewRssService(token),
		SentimentService:  sentiment.NewSentimentService(token),
		SmsService:        sms.NewSmsService(token),
		StockService:      stock.NewStockService(token),
		StreamService:     stream.NewStreamService(token),
		SunnahService:     sunnah.NewSunnahService(token),
		ThumbnailService:  thumbnail.NewThumbnailService(token),
		TimeService:       time.NewTimeService(token),
		TwitterService:    twitter.NewTwitterService(token),
		UrlService:        url.NewUrlService(token),
		UserService:       user.NewUserService(token),
		VehicleService:    vehicle.NewVehicleService(token),
		WeatherService:    weather.NewWeatherService(token),
	}
}

type Client struct {
	token string

	AddressService    *address.AddressService
	AnswerService     *answer.AnswerService
	CacheService      *cache.CacheService
	CryptoService     *crypto.CryptoService
	CurrencyService   *currency.CurrencyService
	DbService         *db.DbService
	EmailService      *email.EmailService
	EmojiService      *emoji.EmojiService
	EvchargersService *evchargers.EvchargersService
	FileService       *file.FileService
	ForexService      *forex.ForexService
	GeocodingService  *geocoding.GeocodingService
	HelloworldService *helloworld.HelloworldService
	HolidaysService   *holidays.HolidaysService
	IdService         *id.IdService
	ImageService      *image.ImageService
	IpService         *ip.IpService
	LocationService   *location.LocationService
	NotesService      *notes.NotesService
	OtpService        *otp.OtpService
	PostcodeService   *postcode.PostcodeService
	PrayerService     *prayer.PrayerService
	QrService         *qr.QrService
	QuranService      *quran.QuranService
	RoutingService    *routing.RoutingService
	RssService        *rss.RssService
	SentimentService  *sentiment.SentimentService
	SmsService        *sms.SmsService
	StockService      *stock.StockService
	StreamService     *stream.StreamService
	SunnahService     *sunnah.SunnahService
	ThumbnailService  *thumbnail.ThumbnailService
	TimeService       *time.TimeService
	TwitterService    *twitter.TwitterService
	UrlService        *url.UrlService
	UserService       *user.UserService
	VehicleService    *vehicle.VehicleService
	WeatherService    *weather.WeatherService
}
