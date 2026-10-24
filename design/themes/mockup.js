// Draws the mockup screens. Every theme page defines window.THEME (name,
// description and a class string for each UI piece) and then loads this file,
// so all themes show exactly the same content and only the look differs.
//
// Card pictures come from the linked card database (resources/duelmasters_db).
// Open the pages straight from disk.

const IMG = "../../resources/duelmasters_db/data/images/";

const CARD = {
  bolshack: { id: "0079", name: "Bolshack Dragon", civ: "fire" },
  hulcus: { id: "0033", name: "Aqua Hulcus", civ: "water" },
  bronze: { id: "0100", name: "Bronze-Arm Tribe", civ: "nature" },
  deathliger: { id: "0005", name: "Deathliger, Lion of Chaos", civ: "darkness" },
  hanusa: { id: "0001", name: "Hanusa, Radiance Elemental", civ: "light" },
  gatling: { id: "1372", name: "Gatling Cyclone", civ: "fire" },
  surfer: { id: "0321", name: "Aqua Surfer", civ: "water" },
  terror: { id: "0073", name: "Terror Pit", civ: "darkness" },
  awe: { id: "0016", name: "Holy Awe", civ: "light" },
  snare: { id: "0109", name: "Natural Snare", civ: "nature" },
  zyler: { id: "0080", name: "Brawler Zyler", civ: "fire" },
  bone: { id: "0058", name: "Bone Spider", civ: "darkness" },
  gigaberos: { id: "0065", name: "Gigaberos", civ: "darkness" },
  vehicle: { id: "0036", name: "Aqua Vehicle", civ: "water" },
  memory: { id: "0039", name: "Crystal Memory", civ: "water" },
  tornado: { id: "0098", name: "Tornado Flame", civ: "fire" },
};

// The five civilizations' colours are the same in every theme.
const CIV_DOT = {
  light: "bg-yellow-300",
  water: "bg-blue-500",
  darkness: "bg-zinc-900 ring-1 ring-zinc-400/70",
  fire: "bg-red-500",
  nature: "bg-green-500",
};

const TIER_DOT = {
  Bronze: "bg-amber-700",
  Silver: "bg-slate-300",
  Gold: "bg-yellow-400",
  Platinum: "bg-cyan-300",
  Diamond: "bg-violet-300",
};

const GAMES = [
  { name: "Kettou Da!", host: "Shobu", tier: "Diamond", rating: 1742, format: "DM-01 to DM-12", civs: ["fire", "nature"], status: "waiting", watching: 0 },
  { name: "Friday Duel", host: "blankscreen", tier: "Gold II", rating: 1482, format: "Open", civs: ["water", "darkness"], status: "playing", watching: 3 },
  { name: "Speed attackers only", host: "Kokujo", tier: "Platinum I", rating: 1688, format: "Open", civs: ["darkness"], status: "playing", watching: 7 },
  { name: "New player, be kind", host: "Mimi", tier: "Bronze III", rating: 962, format: "DM-01 to DM-04", civs: ["light", "water"], status: "waiting", watching: 0 },
  { name: "Rematch?", host: "Rekuta", tier: "Silver I", rating: 1281, format: "DM-01 to DM-12", civs: ["fire"], status: "waiting", watching: 1 },
  { name: "Control mirror", host: "Hakuoh", tier: "Diamond", rating: 1810, format: "Open", civs: ["light", "darkness"], status: "playing", watching: 12 },
];

const ONLINE = [
  { name: "Shobu", tier: "Diamond", status: "In lobby" },
  { name: "Hakuoh", tier: "Diamond", status: "In a game" },
  { name: "Kokujo", tier: "Platinum I", status: "In a game" },
  { name: "Rekuta", tier: "Silver I", status: "Building a deck" },
  { name: "Mimi", tier: "Bronze III", status: "In lobby" },
];

const LOBBY_CHAT = [
  { who: "Shobu", text: "Anyone up for a DM-01 to DM-12 duel?" },
  { who: "Mimi", text: "Is Aqua Surfer worth it in a rush deck?" },
  { system: "Hakuoh won against Kokujo in 14 turns." },
  { who: "Rekuta", text: "Bolshack decks are back!" },
];

const T = window.THEME.T;
const cls = (...parts) => parts.filter(Boolean).join(" ");

function tierName(tier) {
  return tier.split(" ")[0];
}

function tierBadge(tier, rating) {
  return `<span class="${T.tier}"><span class="h-2 w-2 rounded-full ${TIER_DOT[tierName(tier)]}"></span>${tier}${rating ? ` · ${rating}` : ""}</span>`;
}

function civDots(civs, size = "h-2.5 w-2.5") {
  return `<span class="inline-flex gap-1">${civs.map((civ) => `<span class="${size} rounded-full ${CIV_DOT[civ]}" title="${civ}"></span>`).join("")}</span>`;
}

function cardImage(card, size, extra = "") {
  return `<div class="${cls(size, "aspect-[491/682] shrink-0 overflow-hidden rounded-[5%] bg-black/40 shadow-md", extra)}" title="${card.name}">
    <img src="${IMG}${card.id}.webp" alt="${card.name}" class="h-full w-full object-cover" onerror="this.style.visibility='hidden'">
  </div>`;
}

// look is the theme's card back, or its shield for a card in the shield zone.
function cardBack(size, look = T.cardBack) {
  return `<div class="${cls(size, "aspect-[491/682] shrink-0 rounded-[5%]", look)}"></div>`;
}

function panel(title, civ, body, extra = "") {
  return `<section class="${cls(T.panel, T.panelCiv?.[civ], extra)}">
    <header class="${T.panelHead}"><h2 class="${T.panelTitle}">${title}</h2></header>
    <div class="p-4">${body}</div>
  </section>`;
}

function header(active, aside = "") {
  const nav = ["Lobby", "Decks", "Rankings", "Rules"]
    .map((item) => `<a href="#" class="${item === active ? T.navActive : T.nav}">${item}</a>`)
    .join("");

  return `<header class="${T.header}">
    <div class="flex items-center gap-6">
      <a href="#" class="flex items-center gap-2"><span class="${T.logoMark}"></span><span class="${T.logo}">Duel Masters</span></a>
      <nav class="hidden items-center gap-1 md:flex">${nav}</nav>
    </div>
    <div class="flex items-center gap-3">
      ${aside}
      <div class="hidden text-right sm:block">
        <div class="text-sm font-semibold">blankscreen</div>
        ${tierBadge("Gold II", 1482)}
      </div>
      <span class="${T.avatar}">B</span>
    </div>
  </header>`;
}

function gamesTable() {
  const rows = GAMES.map(
    (game) => `<tr class="${T.row}">
      <td class="px-3 py-2.5">
        <div class="font-semibold">${game.name}</div>
        <div class="mt-0.5 flex items-center gap-2 text-xs ${T.subtle}">${civDots(game.civs)}<span>${game.format}</span></div>
      </td>
      <td class="hidden px-3 py-2.5 sm:table-cell"><div class="text-sm">${game.host}</div>${tierBadge(game.tier, game.rating)}</td>
      <td class="px-3 py-2.5"><span class="${game.status === "waiting" ? T.pillWaiting : T.pillPlaying}">${game.status === "waiting" ? "Waiting" : "In game"}</span></td>
      <td class="hidden px-3 py-2.5 text-sm md:table-cell ${T.subtle}">${game.watching ? `${game.watching} watching` : "-"}</td>
      <td class="px-3 py-2.5 text-right">${game.status === "waiting" ? `<button class="${T.btnSmall}">Join</button>` : `<button class="${T.btnSmallAlt}">Watch</button>`}</td>
    </tr>`,
  ).join("");

  return `<div class="overflow-x-auto"><table class="w-full text-left">
    <thead><tr class="${T.tableHead}"><th class="px-3 py-2">Game</th><th class="hidden px-3 py-2 sm:table-cell">Host</th><th class="px-3 py-2">Status</th><th class="hidden px-3 py-2 md:table-cell">Spectators</th><th class="px-3 py-2"></th></tr></thead>
    <tbody>${rows}</tbody>
  </table></div>`;
}

function createGameForm() {
  const choice = (options, chosen) =>
    `<div class="flex gap-1">${options.map((option) => `<button class="${option === chosen ? T.toggleOn : T.toggleOff}">${option}</button>`).join("")}</div>`;

  return `<div class="space-y-4">
    <label class="block"><span class="${T.label}">Game name</span><input class="${T.input}" value="Kettou Da!"></label>
    <div>
      <span class="${T.label}">Deck</span>
      <div class="${cls(T.input, "flex items-center gap-3")}">
        ${cardImage(CARD.bolshack, "w-9")}
        <div class="min-w-0 flex-1"><div class="truncate font-semibold">Bolshack Rush</div><div class="flex items-center gap-2 text-xs ${T.subtle}">${civDots(["fire", "nature"])}40 cards</div></div>
        <span class="${T.subtle}">▾</span>
      </div>
    </div>
    <div><span class="${T.label}">Format</span>${choice(["Open", "DM-01 to DM-12"], "DM-01 to DM-12")}</div>
    <div><span class="${T.label}">Visibility</span>${choice(["Public", "Private"], "Public")}</div>
    <label class="flex items-center gap-2 text-sm"><input type="checkbox" checked class="h-4 w-4"> Allow spectators</label>
    <button class="${cls(T.btn, "w-full")}">Create game</button>
  </div>`;
}

function onlineList() {
  return `<ul class="space-y-2.5">${ONLINE.map(
    (player) => `<li class="flex items-center justify-between gap-3">
      <div class="flex items-center gap-2.5"><span class="${T.avatarSmall}">${player.name[0]}</span><div><div class="text-sm font-semibold">${player.name}</div>${tierBadge(player.tier)}</div></div>
      <span class="text-xs ${T.subtle}">${player.status}</span>
    </li>`,
  ).join("")}</ul>`;
}

function lobbyChat() {
  const lines = LOBBY_CHAT.map((line) =>
    line.system
      ? `<p class="${T.chatSystem}">${line.system}</p>`
      : `<p class="text-sm"><span class="${T.chatName}">${line.who}:</span> ${line.text}</p>`,
  ).join("");

  return `<div class="space-y-2">${lines}</div>
    <div class="mt-4 flex gap-2"><input class="${cls(T.input, "mt-0")}" placeholder="Say something to the lobby"><button class="${T.btnAlt}">Send</button></div>`;
}

function lobby() {
  return `${header("Lobby")}
  <main class="mx-auto grid max-w-7xl gap-6 px-4 py-6 lg:grid-cols-[1fr_22rem]">
    <div class="space-y-6">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div><h1 class="${T.pageTitle}">Lobby</h1><p class="${T.subtle}">Join a waiting game, watch one, or create your own.</p></div>
        <div class="flex gap-2"><button class="${T.btnAlt}">Quick match</button><button class="${T.btn} lg:hidden">Create game</button></div>
      </div>
      ${panel("Open games", "water", gamesTable(), "overflow-hidden")}
      ${panel("Lobby chat", "light", lobbyChat())}
    </div>
    <aside class="space-y-6">
      ${panel("Create a game", "fire", createGameForm())}
      ${panel("Online now · 5", "nature", onlineList())}
    </aside>
  </main>`;
}

function zone(label, content, extra = "") {
  return `<div class="${cls("flex min-w-0 flex-col gap-1", extra)}"><span class="${T.zoneLabel}">${label}</span><div class="flex min-w-0 items-end gap-1.5">${content}</div></div>`;
}

function playerBar(name, tier, rating, deckName, civs, mine) {
  return `<div class="${T.playerBar}">
    <div class="flex items-center gap-3">
      <span class="${T.avatar}">${name[0].toUpperCase()}</span>
      <div><div class="${T.playerName}">${name}${mine ? ` <span class="${T.subtle} text-xs font-normal">(you)</span>` : ""}</div>${tierBadge(tier, rating)}</div>
    </div>
    <div class="flex items-center gap-2 text-sm ${T.subtle}">${civDots(civs)}<span class="hidden sm:inline">${deckName}</span></div>
  </div>`;
}

function phaseBar() {
  const phases = ["Draw", "Mana", "Summon", "Attack", "End"]
    .map((phase, index) => `<span class="${index < 2 ? T.phaseDone : index === 2 ? T.phaseActive : T.phase}">${phase}</span>`)
    .join("");

  return `<div class="flex items-center justify-center gap-2">
    <span class="${T.turnPill}">Turn <b>7</b></span>
    <div class="${T.phaseBar}">${phases}</div>
  </div>`;
}

function gameLog() {
  const mine = (card) => `<span class="${T.logMine}">${card.name}</span>`;
  const theirs = (card) => `<span class="${T.logTheirs}">${card.name}</span>`;
  const lines = [
    `Your turn, blankscreen`,
    `blankscreen summoned ${mine(CARD.bolshack)} to the battle zone`,
    `${theirs(CARD.bone)} (5000) was destroyed by ${mine(CARD.bolshack)} (11000)`,
    `2 of Shobu's shields were broken`,
    `Shobu casted the spell ${theirs(CARD.terror)}`,
    `${mine(CARD.hulcus)} was destroyed by ${theirs(CARD.terror)}`,
  ];

  return `<div class="flex h-full flex-col">
    <div class="${T.logHead}">Game log</div>
    <div class="flex-1 space-y-1.5 overflow-hidden p-3 text-xs leading-snug">${lines.map((line) => `<p>${line}</p>`).join("")}</div>
    <div class="p-3"><input class="${cls(T.input, "mt-0 text-xs")}" placeholder="Chat"></div>
  </div>`;
}

function board() {
  const shields = (count) => Array.from({ length: count }, () => cardBack("w-10 sm:w-12", T.shield)).join("");  const backs = (count) => Array.from({ length: count }, () => cardBack("w-8 sm:w-10")).join("");
  const mana = (cards, used) => cards.map((card, index) => cardImage(card, "w-8 sm:w-10", index < used ? "grayscale brightness-50" : "")).join("");

  return `<div class="${T.mat}">
    <div class="flex min-h-0 flex-1 flex-col justify-between gap-3 p-3 sm:p-4">
      <div class="flex flex-wrap items-end justify-between gap-3">
        ${zone("Hand · 4", backs(4))}
        ${zone("Mana · 5", mana([CARD.bone, CARD.terror, CARD.gigaberos, CARD.deathliger, CARD.bone], 5))}
        ${zone("Shields · 3", shields(3))}
      </div>
      ${zone("Battle zone", cardImage(CARD.gigaberos, "w-14 sm:w-16") + cardImage(CARD.deathliger, "w-14 sm:w-16"), cls("items-center", T.battleRow))}
      ${phaseBar()}
      ${zone("Battle zone", cardImage(CARD.bolshack, "w-14 sm:w-16", T.attacking) + cardImage(CARD.bronze, "w-14 sm:w-16") + cardImage(CARD.hulcus, "w-14 sm:w-16"), cls("items-center", T.battleRow))}
      <div class="flex flex-wrap items-end justify-between gap-3">
        ${zone("Shields · 5", shields(5))}
        ${zone("Mana · 6", mana([CARD.zyler, CARD.snare, CARD.tornado, CARD.gatling, CARD.bronze, CARD.bolshack], 4))}
      </div>
      <div class="flex justify-center gap-1.5">${[CARD.gatling, CARD.surfer, CARD.snare, CARD.tornado, CARD.zyler].map((card) => cardImage(card, "w-12 sm:w-16")).join("")}</div>
    </div>
    <aside class="${T.logPanel}">${gameLog()}</aside>
  </div>`;
}

function game() {
  const aside = `<div class="hidden items-center gap-2 lg:flex"><span class="${T.chip}">Friday Duel</span><span class="${T.chip}">DM-01 to DM-12</span><span class="${T.chip}">3 watching</span></div>`;

  return `${header("Lobby", aside)}
  <main class="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-4">
    ${playerBar("Shobu", "Diamond", 1742, "Darkness Control", ["darkness"], false)}
    ${board()}
    ${playerBar("blankscreen", "Gold II", 1482, "Bolshack Rush", ["fire", "nature"], true)}
    <div class="flex justify-end gap-2"><button class="${T.btnAlt}">Leave</button><button class="${T.btnDanger}">Surrender</button></div>
    <p class="text-center text-xs ${T.subtle}">The board is drawn by the duel board component; this mockup only shows how the theme frames and tints it.</p>
  </main>`;
}

// A neutral switcher outside the theme, for moving between screens and themes.
function switcher(screen) {
  const link = (target, label) =>
    `<a href="#${target}" class="rounded-full px-3 py-1 ${screen === target ? "bg-white text-black" : "text-white/80 hover:text-white"}">${label}</a>`;

  return `<div class="fixed bottom-4 right-4 z-50 flex items-center gap-1 rounded-full bg-black/80 p-1 text-xs font-sans shadow-xl ring-1 ring-white/20 backdrop-blur">
    <a href="index.html" class="px-3 py-1 text-white/60 hover:text-white">← Themes</a>
    ${link("lobby", "Lobby")}${link("game", "Game")}
  </div>`;
}

function render() {
  const screen = location.hash === "#game" ? "game" : "lobby";
  document.title = `Duel Masters · ${window.THEME.name} · ${screen === "game" ? "Game" : "Lobby"}`;
  document.getElementById("app").innerHTML = (screen === "game" ? game() : lobby()) + switcher(screen);
}

window.addEventListener("hashchange", render);
render();
