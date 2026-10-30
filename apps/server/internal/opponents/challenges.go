package opponents

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// Pause is the admin setting for how long the computer waits before each move.
var Pause = settings.Number{
	Key:         "computer.pause_milliseconds",
	Group:       "Computer opponents",
	Description: "How long, in milliseconds, the computer waits before each move, so a player can follow what it does. Applies to games started afterwards.",
	Default:     800,
	Min:         0,
	Max:         5000,
}

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{Pause}
}

// ErrUnavailable is returned for an opponent that cannot play for now: it is
// no longer offered, or its deck no longer keeps to its rules.
var ErrUnavailable = errors.New("this computer opponent cannot play right now")

// Offer is a computer opponent as players are offered it.
type Offer struct {
	Opponent
	FormatName string
	// Civilizations are those of its deck, in the game's order.
	Civilizations []string
}

// ChallengeDependencies are what games against the computer need.
type ChallengeDependencies struct {
	Opponents *Service
	Formats   *formats.Service
	Lobby     *lobby.Service
	Games     *games.Service
	// Library is what the computer knows of the cards.
	Library  *computer.Library
	Settings *settings.Service
	Logger   *slog.Logger
}

// Challenges starts players' games against the computer opponents. These games
// are private, never count, and are started like any other, so the games
// service limits, runs and records them. It is safe for concurrent use.
type Challenges struct {
	opponents *Service
	formats   *formats.Service
	lobby     *lobby.Service
	games     *games.Service
	library   *computer.Library
	settings  *settings.Service
	logger    *slog.Logger
}

// NewChallenges returns games against the computer.
func NewChallenges(deps ChallengeDependencies) *Challenges {
	return &Challenges{
		opponents: deps.Opponents,
		formats:   deps.Formats,
		lobby:     deps.Lobby,
		games:     deps.Games,
		library:   deps.Library,
		settings:  deps.Settings,
		logger:    deps.Logger,
	}
}

// Offers lists the opponents players are offered, in order, leaving out any
// that cannot play for now: a format's ban list may have changed since its deck
// was made.
func (c *Challenges) Offers(ctx context.Context) ([]Offer, error) {
	list, err := c.opponents.List(ctx)
	if err != nil {
		return nil, err
	}

	offers := make([]Offer, 0, len(list))
	for _, opponent := range list {
		problems, err := c.opponents.Check(ctx, opponent)
		if err != nil {
			return nil, err
		}
		if len(problems) > 0 {
			continue
		}

		format, err := c.formats.ByID(ctx, opponent.FormatID)
		if err != nil {
			return nil, fmt.Errorf("finding %s's format: %w", opponent.Name, err)
		}
		civilizations, err := c.games.Civilizations(ctx, opponent.Deck())
		if err != nil {
			return nil, fmt.Errorf("reading %s's deck: %w", opponent.Name, err)
		}
		offers = append(offers, Offer{Opponent: opponent, FormatName: format.Name, Civilizations: civilizations})
	}
	return offers, nil
}

// Start starts a game between a player, with one of their decks legal in the
// opponent's format, and a computer opponent. The player hosts, and goes to the
// game with the id returned; it begins with the coin toss. It returns
// ErrNotFound for an opponent not offered, ErrUnavailable for one that cannot
// play, lobby.ErrPaused while new games are paused, a *lobby.BusyError for a
// player already doing something, games.ErrComputerGamesFull while as many
// such games as allowed are running, and the errors
// lobby.Service.PrepareDeck returns for the player's deck.
func (c *Challenges) Start(ctx context.Context, player users.User, opponentID, deckID string) (games.Game, error) {
	if c.lobby.Paused() {
		return games.Game{}, lobby.ErrPaused
	}

	opponent, err := c.opponents.Get(ctx, opponentID)
	if err != nil {
		return games.Game{}, err
	}
	format, deck, err := c.lobby.PrepareDeck(ctx, player, opponent.FormatID, deckID)
	if err != nil {
		return games.Game{}, err
	}
	seat, err := c.seat(ctx, opponent)
	if err != nil {
		return games.Game{}, err
	}

	// Claimed first, so a player cannot start two at once.
	activities := c.lobby.Activities()
	claim := uuid.NewString()
	if err := activities.Claim(player.ID, lobby.Activity{Kind: lobby.Playing, GameID: claim, Computer: true}); err != nil {
		return games.Game{}, err
	}

	game, err := c.games.Start(ctx, games.StartInput{
		Name:   opponent.Name,
		Format: format,
		Host:   games.Entrant{User: player, Deck: deck},
		Guest:  seat,
	})
	if err != nil {
		activities.Release(player.ID, lobby.Playing, claim)

		var illegal *games.IllegalDeckError
		if errors.As(err, &illegal) && illegal.Username == opponent.Name {
			return games.Game{}, ErrUnavailable
		}
		return games.Game{}, err
	}

	activities.Change(player.ID, lobby.Playing, lobby.Activity{Kind: lobby.Playing, GameID: game.ID, Computer: true})
	return game, nil
}

// Entrant is a computer opponent's seat in a game, by its account. It returns
// ErrUnavailable for an account of an opponent that cannot play.
func (c *Challenges) Entrant(ctx context.Context, userID string) (games.Entrant, error) {
	opponent, err := c.opponents.ByUser(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return games.Entrant{}, ErrUnavailable
	}
	if err != nil {
		return games.Entrant{}, err
	}
	return c.seat(ctx, opponent)
}

// seat is the opponent's seat in a game, once its deck is checked. An admin
// keeps a deck to its rules, but a format's ban list may have changed since.
func (c *Challenges) seat(ctx context.Context, opponent Opponent) (games.Entrant, error) {
	problems, err := c.opponents.Check(ctx, opponent)
	if err != nil {
		return games.Entrant{}, err
	}
	if len(problems) > 0 {
		c.logger.WarnContext(ctx, "a computer opponent's deck cannot be played", "opponent", opponent.Name, "problem", problems[0].Message())
		return games.Entrant{}, ErrUnavailable
	}

	style, ok := computer.StyleNamed(opponent.Style)
	if !ok {
		return games.Entrant{}, fmt.Errorf("computer opponent %q has an unknown style %q", opponent.Name, opponent.Style)
	}

	return games.Entrant{
		User: users.User{ID: opponent.UserID, Username: opponent.Name, Computer: true},
		Deck: opponent.Deck(),
		Computer: &computer.Options{
			Library:  c.library,
			Style:    style,
			Strength: opponent.Strength,
			Pause:    time.Duration(c.settings.Number(Pause)) * time.Millisecond,
		},
	}, nil
}
