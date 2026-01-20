package usecase

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// Indonesian nouns for persona generation
var nouns = []string{
	// Animals & Creatures
	"Kucing", "Ayam", "Bebek", "Angsa", "Soang", "Kambing", "Sapi", "Kerbau",
	"Kuda", "Kelelawar", "Tupai", "Marmut", "Hamster", "Kelinci", "Musang",
	"Biawak", "Tokek", "Cicak", "Kadal", "Komodo", "Buaya", "Ular", "Kura-kura",
	"Lele", "Gurame", "Mujaer", "Teri", "Kakap", "Tongkol", "Hiu", "Paus",
	"Kecoak", "Semut", "Nyamuk", "Lalat", "Lebah", "Tawon", "Belalang", "Jangkrik",
	"Kupu-kupu", "Ulat", "Cacing", "Bekicot", "Keong", "Undur-undur", "Kepiting",
	"Monyet", "Lutung", "Gorila", "Orangutan", "Kukang", "Panda", "Koala", "Gajah",
	"Pocong", "Kunti", "Tuyul", "Genderuwo", "Jenglot", "Wewe", "Buto", "Barongan",

	// Household Items
	"Kulkas", "Ricecooker", "Jemuran", "Odong-odong", "Sepeda", "Becak", "Bajaj",
	"Kompor", "Setrika", "Meja", "Kursi", "Panci", "Wajan", "Spatula", "Cobek", "Ulekan",
	"Sendok", "Garpu", "Guling", "Bantal", "Kasur", "Tikar", "Karpet", "Keset",
	"Sapu", "Pel", "Ember", "Gayung", "Sikat", "Sabun", "Sampo", "Odol",
	"Kipas", "AC", "TV", "Radio", "Laptop", "HP", "Charger", "Kabel", "Colokan",
	"Galon", "Dispenser", "Termos", "Teko", "Gelas", "Piring", "Mangkok", "Botol",
	"Lemari", "Laci", "Pintu", "Jendela", "Genteng", "Pagar", "Tiang", "Lampu",

	// Food & Snacks
	"Kerupuk", "Kemplang", "Rengginang", "Peyek", "Emping", "Opak", "Kecimpring",
	"Cilok", "Cireng", "Cimol", "Batagor", "Siomay", "Otak-otak", "Pempek", "Tekwan",
	"Bakso", "Mie", "Soto", "Sate", "Gule", "Rawon", "Rendang", "Gudeg",
	"Tahu", "Tempe", "Oncom", "Menjes", "Gembus", "Bacem", "Pepes", "Botok",
	"Lontong", "Ketupat", "Lemper", "Arem-arem", "SemarMendem", "Nagasari",
	"Dodol", "Wajik", "Klepon", "Onde-onde", "Cenenil", "Getuk", "Tiwul", "Gatot",
	"Martabak", "TerangBulan", "Pukis", "Cubit", "Pancong", "Bandros", "Serabi",

	// Random Objects
	"Sandal", "Sepatu", "Kaos", "Kemeja", "Celana", "Kolor", "Sarung", "Peci",
	"Helm", "Jaket", "JasHujan", "Payung", "Kacamata", "Topi", "Dasi", "Sabuk",
	"Tas", "Koper", "Dompet", "Kunci", "Gembok", "Rantai", "Paku", "Palu", "Obeng",
	"Ban", "Velg", "Stang", "Spion", "Knalpot", "Busi", "Aki", "Bensin", "Solar",
	"Batu", "Bata", "Pasir", "Semen", "Kayu", "Bambu", "Rotan", "Daun", "Ranting",
}

// Indonesian adjectives/fun verbs for persona generation
var adjectives = []string{
	// Actions & Movements
	"Kayang", "Koprol", "Salto", "Melintir", "Ngesot", "Merayap", "Mencolot", "Terbang",
	"Lari", "Jalan", "Duduk", "Jongkok", "Tengkurap", "Telentang", "Nungging", "Nyungsep",
	"Joget", "Dugem", "Goyang", "Dangdutan", "Headbang", "Breakdance", "Shuffle",
	"Nafas", "Ngorok", "Henyak", "Melamun", "Bengong", "Mikir", "Pusing", "Pening",

	// Funny States & Traits
	"Kocak", "Gokil", "Nyentrik", "Konyol", "Semriwing", "Kepo", "Baper", "Caper",
	"Sotoy", "Julid", "Alay", "Lebay", "Kuper", "Kudet", "Gabut", "Mager", "Santuy", "Woles",
	"Lincah", "Geli", "Kece", "Maknyus", "Ngakak", "Ambyar", "Sambat", "Galau",
	"Licin", "Lentur", "Ceria", "Aneh", "Gembul", "Mblebek", "Tembem", "Cemong",
	"Gesit", "Menyenggol", "Kilat", "Gagah", "Cekrek", "Garing", "Ngocol",
	"Gemoy", "Imut", "Lucu", "Sangar", "Galak", "Judes", "Sinister", "Menyeramkan",
	"Demes", "Lugu", "Polos", "Riang", "Ringan", "Berat", "Kuat", "Lemah",
	"Berani", "Nakal", "Bandels", "Luwes", "Ngebut", "Lambat", "Kalem", "Rusuh",

	// Slang & Random
	"Meleyot", "Mlehoy", "Menyala", "Abangku", "Suhu", "Sepuh", "Newbie", "Noob", "Pro",
	"Cihuy", "Eaaa", "Yoi", "Gaskeun", "Skuy", "Otw", "Mabar", "Nobar",
	"Sultan", "Misqueen", "Halu", "Gengsi", "Pansos", "FOMO", "Jomblo", "Bucin",
	"Gondrong", "Botak", "Kribo", "Poni", "Mancung", "Pesek", "Jenong",
	"Kedinginan", "Kepanasan", "MasukAngin", "Kerokan", "Pegel", "Linu", "Kesemutan",
}

// Neon colors for personas
var neonColors = []string{
	"#FFD100", // Kuning Neon
	"#FF6AC1", // Pink Neon
	"#00E676", // Stabilo Hijau
	"#00E5FF", // Cyan Neon
	"#FF5252", // Merah Neon
	"#B388FF", // Ungu Neon
	"#FF9100", // Orange Neon
	"#69F0AE", // Mint Neon
}

// PersonaGenerator generates unique personas for users
type PersonaGenerator struct {
	mu       sync.RWMutex
	existing map[string]bool
}

// NewPersonaGenerator creates a new PersonaGenerator
func NewPersonaGenerator() *PersonaGenerator {
	return &PersonaGenerator{
		existing: make(map[string]bool),
	}
}

// Generate creates a unique persona name and color
func (pg *PersonaGenerator) Generate() *domain.User {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	var name string
	maxAttempts := 100

	for i := 0; i < maxAttempts; i++ {
		noun := nouns[rand.Intn(len(nouns))]
		adj := adjectives[rand.Intn(len(adjectives))]
		name = fmt.Sprintf("%s %s", noun, adj)

		if !pg.existing[name] {
			break
		}

		// Add suffix if still duplicate after max attempts
		if i == maxAttempts-1 {
			name = fmt.Sprintf("%s %d", name, rand.Intn(999))
		}
	}

	pg.existing[name] = true
	color := neonColors[rand.Intn(len(neonColors))]

	return domain.NewUser(name, color)
}

// GenerateWithPersona creates a user with an existing persona name and color
// Used when a user reconnects/refreshes and wants to keep their persona
func (pg *PersonaGenerator) GenerateWithPersona(name string, color string) *domain.User {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	// Mark this persona as existing
	pg.existing[name] = true

	return domain.NewUser(name, color)
}

// Release removes a persona from the active set
func (pg *PersonaGenerator) Release(name string) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	delete(pg.existing, name)
}

// ActiveCount returns the number of active personas
func (pg *PersonaGenerator) ActiveCount() int {
	pg.mu.RLock()
	defer pg.mu.RUnlock()
	return len(pg.existing)
}
