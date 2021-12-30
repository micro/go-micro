package services

import (
	"go-micro.dev/v4/services/address"
	"go-micro.dev/v4/services/answer"
	"go-micro.dev/v4/services/app"
	"go-micro.dev/v4/services/avatar"
	"go-micro.dev/v4/services/cache"
	"go-micro.dev/v4/services/contact"
	"go-micro.dev/v4/services/crypto"
	"go-micro.dev/v4/services/currency"
	"go-micro.dev/v4/services/db"
	"go-micro.dev/v4/services/email"
	"go-micro.dev/v4/services/emoji"
	"go-micro.dev/v4/services/evchargers"
	"go-micro.dev/v4/services/event"
	"go-micro.dev/v4/services/file"
	"go-micro.dev/v4/services/forex"
	"go-micro.dev/v4/services/function"
	"go-micro.dev/v4/services/geocoding"
	"go-micro.dev/v4/services/gifs"
	"go-micro.dev/v4/services/google"
	"go-micro.dev/v4/services/helloworld"
	"go-micro.dev/v4/services/holidays"
	"go-micro.dev/v4/services/id"
	"go-micro.dev/v4/services/image"
	"go-micro.dev/v4/services/ip"
	"go-micro.dev/v4/services/joke"
	"go-micro.dev/v4/services/location"
	"go-micro.dev/v4/services/movie"
	"go-micro.dev/v4/services/mq"
	"go-micro.dev/v4/services/news"
	"go-micro.dev/v4/services/nft"
	"go-micro.dev/v4/services/notes"
	"go-micro.dev/v4/services/otp"
	"go-micro.dev/v4/services/postcode"
	"go-micro.dev/v4/services/prayer"
	"go-micro.dev/v4/services/qr"
	"go-micro.dev/v4/services/quran"
	"go-micro.dev/v4/services/routing"
	"go-micro.dev/v4/services/rss"
	"go-micro.dev/v4/services/search"
	"go-micro.dev/v4/services/sentiment"
	"go-micro.dev/v4/services/sms"
	"go-micro.dev/v4/services/space"
	"go-micro.dev/v4/services/spam"
	"go-micro.dev/v4/services/stock"
	"go-micro.dev/v4/services/stream"
	"go-micro.dev/v4/services/sunnah"
	"go-micro.dev/v4/services/thumbnail"
	"go-micro.dev/v4/services/time"
	"go-micro.dev/v4/services/translate"
	"go-micro.dev/v4/services/twitter"
	"go-micro.dev/v4/services/url"
	"go-micro.dev/v4/services/user"
	"go-micro.dev/v4/services/vehicle"
	"go-micro.dev/v4/services/weather"
	"go-micro.dev/v4/services/youtube"
)

func NewClient(token string) *Client {
	return &Client{
		token: token,

		AddressService:    address.NewAddressService(token),
		AnswerService:     answer.NewAnswerService(token),
		AppService:        app.NewAppService(token),
		AvatarService:     avatar.NewAvatarService(token),
		CacheService:      cache.NewCacheService(token),
		ContactService:    contact.NewContactService(token),
		CryptoService:     crypto.NewCryptoService(token),
		CurrencyService:   currency.NewCurrencyService(token),
		DbService:         db.NewDbService(token),
		EmailService:      email.NewEmailService(token),
		EmojiService:      emoji.NewEmojiService(token),
		EvchargersService: evchargers.NewEvchargersService(token),
		EventService:      event.NewEventService(token),
		FileService:       file.NewFileService(token),
		ForexService:      forex.NewForexService(token),
		FunctionService:   function.NewFunctionService(token),
		GeocodingService:  geocoding.NewGeocodingService(token),
		GifsService:       gifs.NewGifsService(token),
		GoogleService:     google.NewGoogleService(token),
		HelloworldService: helloworld.NewHelloworldService(token),
		HolidaysService:   holidays.NewHolidaysService(token),
		IdService:         id.NewIdService(token),
		ImageService:      image.NewImageService(token),
		IpService:         ip.NewIpService(token),
		JokeService:       joke.NewJokeService(token),
		LocationService:   location.NewLocationService(token),
		MovieService:      movie.NewMovieService(token),
		MqService:         mq.NewMqService(token),
		NewsService:       news.NewNewsService(token),
		NftService:        nft.NewNftService(token),
		NotesService:      notes.NewNotesService(token),
		OtpService:        otp.NewOtpService(token),
		PostcodeService:   postcode.NewPostcodeService(token),
		PrayerService:     prayer.NewPrayerService(token),
		QrService:         qr.NewQrService(token),
		QuranService:      quran.NewQuranService(token),
		RoutingService:    routing.NewRoutingService(token),
		RssService:        rss.NewRssService(token),
		SearchService:     search.NewSearchService(token),
		SentimentService:  sentiment.NewSentimentService(token),
		SmsService:        sms.NewSmsService(token),
		SpaceService:      space.NewSpaceService(token),
		SpamService:       spam.NewSpamService(token),
		StockService:      stock.NewStockService(token),
		StreamService:     stream.NewStreamService(token),
		SunnahService:     sunnah.NewSunnahService(token),
		ThumbnailService:  thumbnail.NewThumbnailService(token),
		TimeService:       time.NewTimeService(token),
		TranslateService:  translate.NewTranslateService(token),
		TwitterService:    twitter.NewTwitterService(token),
		UrlService:        url.NewUrlService(token),
		UserService:       user.NewUserService(token),
		VehicleService:    vehicle.NewVehicleService(token),
		WeatherService:    weather.NewWeatherService(token),
		YoutubeService:    youtube.NewYoutubeService(token),
	}
}

type Client struct {
	token string

	AddressService    *address.AddressService
	AnswerService     *answer.AnswerService
	AppService        *app.AppService
	AvatarService     *avatar.AvatarService
	CacheService      *cache.CacheService
	ContactService    *contact.ContactService
	CryptoService     *crypto.CryptoService
	CurrencyService   *currency.CurrencyService
	DbService         *db.DbService
	EmailService      *email.EmailService
	EmojiService      *emoji.EmojiService
	EvchargersService *evchargers.EvchargersService
	EventService      *event.EventService
	FileService       *file.FileService
	ForexService      *forex.ForexService
	FunctionService   *function.FunctionService
	GeocodingService  *geocoding.GeocodingService
	GifsService       *gifs.GifsService
	GoogleService     *google.GoogleService
	HelloworldService *helloworld.HelloworldService
	HolidaysService   *holidays.HolidaysService
	IdService         *id.IdService
	ImageService      *image.ImageService
	IpService         *ip.IpService
	JokeService       *joke.JokeService
	LocationService   *location.LocationService
	MovieService      *movie.MovieService
	MqService         *mq.MqService
	NewsService       *news.NewsService
	NftService        *nft.NftService
	NotesService      *notes.NotesService
	OtpService        *otp.OtpService
	PostcodeService   *postcode.PostcodeService
	PrayerService     *prayer.PrayerService
	QrService         *qr.QrService
	QuranService      *quran.QuranService
	RoutingService    *routing.RoutingService
	RssService        *rss.RssService
	SearchService     *search.SearchService
	SentimentService  *sentiment.SentimentService
	SmsService        *sms.SmsService
	SpaceService      *space.SpaceService
	SpamService       *spam.SpamService
	StockService      *stock.StockService
	StreamService     *stream.StreamService
	SunnahService     *sunnah.SunnahService
	ThumbnailService  *thumbnail.ThumbnailService
	TimeService       *time.TimeService
	TranslateService  *translate.TranslateService
	TwitterService    *twitter.TwitterService
	UrlService        *url.UrlService
	UserService       *user.UserService
	VehicleService    *vehicle.VehicleService
	WeatherService    *weather.WeatherService
	YoutubeService    *youtube.YoutubeService
}
