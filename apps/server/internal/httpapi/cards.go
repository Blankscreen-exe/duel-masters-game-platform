package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// pictureCacheControl lets browsers keep pictures and icons for a day, then
// check them again: they only change when the card database is updated.
const pictureCacheControl = "public, max-age=86400"

func (h *handlers) SearchCards(ctx context.Context, request api.SearchCardsRequestObject) (api.SearchCardsResponseObject, error) {
	params := request.Params
	filter := cards.Filter{
		Civilizations: values(params.Civilization),
		Types:         values(params.Type),
		Races:         values(params.Race),
		Keywords:      values(params.Keyword),
		Sets:          values(params.Set),
		Rarities:      values(params.Rarity),
		OnlyIDs:       values(params.CardId),
		ExceptIDs:     values(params.ExceptCardId),
		MinCost:       params.MinCost,
		MaxCost:       params.MaxCost,
		MinPower:      params.MinPower,
		MaxPower:      params.MaxPower,
	}
	if params.Q != nil {
		filter.Query = *params.Q
	}
	if params.Text != nil {
		filter.Text = *params.Text
	}
	if params.CivilizationMatch != nil {
		filter.CivilizationMatch = string(*params.CivilizationMatch)
	}
	if params.CivilizationCount != nil {
		filter.CivilizationCount = string(*params.CivilizationCount)
	}
	if params.Sort != nil {
		filter.Sort = string(*params.Sort)
	}
	if params.Order != nil {
		filter.Descending = *params.Order == api.SearchCardsParamsOrderDesc
	}
	if params.Playable != nil {
		filter.PlayableOnly = *params.Playable
	}
	if params.Page != nil {
		filter.Page = *params.Page
	}
	if params.PageSize != nil {
		filter.PageSize = *params.PageSize
	}

	var err error
	if params.Format != nil {
		filter.Legal, err = h.legalityIn(ctx, *params.Format)
	}
	var page cards.Page
	if err == nil {
		page, err = h.catalogue.Search(ctx, filter)
	}
	if invalid, ok := invalidInput(err, map[string]string{"query": "q"}); ok {
		return api.SearchCards400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	found := make([]api.Card, 0, len(page.Cards))
	for _, card := range page.Cards {
		found = append(found, apiCard(card))
	}

	return api.SearchCards200JSONResponse{Cards: found, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

// legalityIn is what a format allows, for a card search to keep to.
func (h *handlers) legalityIn(ctx context.Context, formatID string) (*cards.Legality, error) {
	format, err := h.formats.ByID(ctx, formatID)
	if errors.Is(err, formats.ErrNotFound) {
		return nil, validation.New("format", "does not exist")
	}
	if err != nil {
		return nil, err
	}

	legal := cards.Legality{AllSets: format.AllSets, Sets: format.Sets}
	for cardID, limit := range format.Limits {
		if limit == 0 {
			legal.Banned = append(legal.Banned, cardID)
		}
	}

	return &legal, nil
}

func (h *handlers) GetCardFilters(ctx context.Context, _ api.GetCardFiltersRequestObject) (api.GetCardFiltersResponseObject, error) {
	filters, err := h.catalogue.FilterValues(ctx)
	if err != nil {
		return nil, err
	}

	sets := make([]api.CardSet, 0, len(filters.Sets))
	for _, set := range filters.Sets {
		entry := api.CardSet{Code: set.Code}
		if set.HasIcon {
			icon := basePath + "/cards/sets/" + url.PathEscape(set.Code) + "/icon"
			entry.IconUrl = &icon
		}
		sets = append(sets, entry)
	}

	rarities := make([]api.CardRarity, 0, len(filters.Rarities))
	for _, rarity := range filters.Rarities {
		rarities = append(rarities, api.CardRarity{Code: rarity.Code, Name: rarity.Name})
	}

	return api.GetCardFilters200JSONResponse{
		Civilizations: filters.Civilizations,
		Types:         filters.Types,
		Races:         filters.Races,
		Keywords:      filters.Keywords,
		Sets:          sets,
		Rarities:      rarities,
	}, nil
}

func (h *handlers) GetCard(ctx context.Context, request api.GetCardRequestObject) (api.GetCardResponseObject, error) {
	card, err := h.catalogue.Card(ctx, request.CardId)
	if errors.Is(err, cards.ErrNotFound) {
		return api.GetCard404JSONResponse{NotFoundJSONResponse: notFound("No such card.")}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetCard200JSONResponse(apiCard(card)), nil
}

func (h *handlers) GetCardImage(ctx context.Context, request api.GetCardImageRequestObject) (api.GetCardImageResponseObject, error) {
	path, err := h.catalogue.ImageFile(ctx, request.Key)
	if errors.Is(err, cards.ErrNotFound) {
		return api.GetCardImage404JSONResponse{NotFoundJSONResponse: notFound("No such picture.")}, nil
	}
	if err != nil {
		return nil, err
	}

	return servedFile{request: requestFrom(ctx), path: path, contentType: "image/webp"}, nil
}

func (h *handlers) GetCardSetIcon(ctx context.Context, request api.GetCardSetIconRequestObject) (api.GetCardSetIconResponseObject, error) {
	path, err := h.catalogue.SetIconFile(ctx, request.SetCode)
	if errors.Is(err, cards.ErrNotFound) {
		return api.GetCardSetIcon404JSONResponse{NotFoundJSONResponse: notFound("No such set icon.")}, nil
	}
	if err != nil {
		return nil, err
	}

	return servedFile{request: requestFrom(ctx), path: path, contentType: "image/png"}, nil
}

// servedFile answers with a file the way net/http's file server does:
// conditional requests answered 304, ranges honoured, and the browser allowed
// to keep it for a while.
type servedFile struct {
	request     *http.Request
	path        string
	contentType string
}

func (f servedFile) VisitGetCardImageResponse(w http.ResponseWriter) error { return f.serve(w) }

func (f servedFile) VisitGetCardSetIconResponse(w http.ResponseWriter) error { return f.serve(w) }

func (f servedFile) serve(w http.ResponseWriter) error {
	file, err := os.Open(f.path)
	if errors.Is(err, os.ErrNotExist) {
		// The catalogue lists the file but it has gone since it was copied.
		writeError(w, http.StatusNotFound, codeNotFound, "No such file.")
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", f.contentType)
	w.Header().Set("Cache-Control", pictureCacheControl)
	http.ServeContent(w, f.request, filepath.Base(f.path), info.ModTime(), file)

	return nil
}

// apiCard is a card as the API shows it.
func apiCard(card cards.Card) api.Card {
	printings := make([]api.CardPrinting, 0, len(card.Printings))
	for _, printing := range card.Printings {
		printings = append(printings, api.CardPrinting{
			Id:              printing.ID,
			Set:             printing.SetCode,
			CollectorNumber: printing.CollectorNumber,
			Rarity:          printing.Rarity,
			RarityCode:      printing.RarityCode,
			ImageUrl:        pictureURL(printing),
		})
	}

	abilities := make([]api.CardAbility, 0, len(card.Abilities))
	for _, ability := range card.Abilities {
		line := api.CardAbility{Text: ability.Text}
		if ability.Reminder != "" {
			reminder := ability.Reminder
			line.Reminder = &reminder
		}
		abilities = append(abilities, line)
	}

	shown := api.Card{
		Id:            card.ID,
		Name:          card.Name,
		Type:          card.Type,
		Cost:          card.Cost,
		Power:         card.Power,
		PowerText:     card.PowerText,
		Civilizations: card.Civilizations,
		Races:         card.Races,
		RulesText:     card.RulesText,
		Abilities:     abilities,
		Keywords:      card.Keywords,
		Categories:    card.Categories,
		Playable:      card.Playable(),
		Printings:     printings,
	}
	if len(printings) > 0 {
		shown.ImageUrl = printings[0].ImageUrl
	}

	return shown
}

// pictureURL is where a printing's picture is served, or nil without one.
func pictureURL(printing cards.Printing) *string {
	if printing.ImagePath == "" {
		return nil
	}

	location := basePath + "/cards/images/" + url.PathEscape(printing.ID)
	return &location
}

func notFound(message string) api.NotFoundJSONResponse {
	return api.NotFoundJSONResponse(apiError(codeNotFound, message))
}

// values is a list query parameter, empty when absent.
func values(list *[]string) []string {
	if list == nil {
		return nil
	}

	return *list
}
