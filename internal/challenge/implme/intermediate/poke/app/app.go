package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mtslzr/pokeapi-go/structs"
	"github.com/romangurevitch/concurrencyworkshop/internal/challenge/implme/intermediate/poke/client"
	"github.com/romangurevitch/concurrencyworkshop/internal/pattern/pubsub"
)

const (
	windowWidth      = 400
	windowHeight     = 600
	defaultURL       = "https://golangify.com/wp-content/uploads/2020/04/go-read.png"
	notFoundImageURL = "https://miro.medium.com/v2/resize:fit:460/1*1Yf_9BPNftL1gdCTMr9Exw.png"
)

type PokeAPP interface {
	Start()
}

type pokeAPP struct {
	pokeClient client.PokeClient

	header *widget.Label
	img    *canvas.Image
	input  *widget.Entry
}

func NewPokeApp(pokeClient client.PokeClient) PokeAPP {
	return &pokeAPP{
		pokeClient: pokeClient,
		header:     createHeader(),
		img:        imageFromURL(defaultURL),
		input:      widget.NewEntry(),
	}
}

type pokeResult struct {
	poke *structs.Pokemon
	err  error
}

var ps = pubsub.NewPubSub[pokeResult]()
var ch = make(chan pubsub.Result[pokeResult], 1)

func (p *pokeAPP) Start() {
	ps.Subscribe("topic1", ch)
	myApp := app.New()
	myWindow := myApp.NewWindow("PokeGUI")
	myWindow.Resize(fyne.NewSize(windowWidth, windowHeight))

	p.input.SetPlaceHolder("e.g., pikachu or 25")
	// TODO: impl and use OnChangedNonBlocking to improve responsiveness of the app
	p.input.OnChanged = p.OnChangedNonBlocking

	content := container.NewStack(container.NewVBox(p.header, p.input), p.img)
	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func (p *pokeAPP) OnChanged(id string) {
	_, err := p.fetchAndUpdatePokemon(id)
	if err != nil {
		slog.Error("fetchAndUpdatePokemon", "error", err)
	}
}

// OnChangedNonBlocking triggers a non-blocking operation to fetch and update
// the Pokémon information based on the provided ID. This method improves the
// responsiveness of the application by not blocking the UI while performing
// the fetch operation. When the ID is entered, this function should be called,
// and it will handle the update asynchronously.
//
// The function is currently not implemented and will panic if used.
// TODO: Implement OnChangedNonBlocking to fetch and update Pokémon details asynchronously.
func (p *pokeAPP) OnChangedNonBlocking(id string) {
	fmt.Println("OnChangedNonBlocking")
	if id == "" {
		p.setImage(defaultURL)
		return
	}
	go func() {
		fmt.Println("OnChangedNonBlocking publish topic1")
		poke, err := p.pokeClient.FetchPokemon(id)
		ps.Publish("topic1", pokeResult{poke: poke, err: err})
	}()

	go func() {
		value, ok := <-ch
		if ok {
			if value.Value.err != nil {
				p.setName("Not Found")
				p.setImage(notFoundImageURL)
			} else if value.Value.poke != nil {
				p.setName(value.Value.poke.Name)
				p.setImage(value.Value.poke.Sprites.FrontDefault)
			}
		}
	}()
}

func (p *pokeAPP) fetchAndUpdatePokemon(id string) (bool, error) {
	if id == "" {
		p.setImage(defaultURL)
		return true, nil
	}

	poke, err := p.pokeClient.FetchPokemon(id)
	if err != nil {
		p.setName("Not Found")
		p.setImage(notFoundImageURL)
		return false, err
	}

	p.setName(poke.Name)
	p.setImage(poke.Sprites.FrontDefault)
	return true, nil
}

func (p *pokeAPP) setName(header string) {
	p.header.SetText("Pokémon: " + header)
}

func (p *pokeAPP) setImage(url string) {
	if !isValidURL(url) {
		url = defaultURL
	}

	p.img.Resource = imageFromURL(url).Resource
	p.img.Refresh()
}

func createHeader() *widget.Label {
	header := widget.NewLabel("Enter Pokémon ID or Name")
	header.TextStyle = fyne.TextStyle{Bold: true}
	return header
}

func imageFromURL(url string) *canvas.Image {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := http.DefaultClient.Do(req)
	defer func() {
		err = response.Body.Close()
		if err != nil {
			slog.Error("Close", "error", err)
		}
	}()

	img := canvas.NewImageFromReader(response.Body, "Pokemon")
	img.FillMode = canvas.ImageFillOriginal

	return img
}

func isValidURL(urlStr string) bool {
	// Parse the URL and ensure there were no errors.
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Check if the URL scheme and host are non-empty to consider this a valid URL.
	// You might want to extend this with more checks depending on your use case.
	return u.Scheme != "" && u.Host != ""
}
