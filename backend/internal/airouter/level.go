package airouter

// levelGuidance is the vocabulary/grammar/passage-length descriptor the
// roadmap prompt hands the model per CEFR level, so a B1 learner gets
// workplace-register B1 content instead of the model's own guess at what
// "B1" means. Passage lengths follow the 1st-thinking placement bank's own
// difficulty ladder (§6.1 asks for progressive difficulty across modules).
var levelGuidance = map[string]string{
	"A1": "A1: use a core vocabulary of roughly 500-1,000 word families — everyday objects, family, numbers, simple routines. Use grammar limited to present simple and present continuous, basic word order, simple yes/no and wh-questions, no subordinate clauses. Reading passages run 60-90 words with short, simple sentences; questions test literal recall, not inference.",
	"A2": "A2: use a vocabulary of roughly 1,000-2,000 word families covering daily life, shopping, travel and work basics. Use grammar limited to past simple, future with 'going to', comparatives and superlatives, simple connectors (because, but, so, when). Reading passages run 90-130 words; questions test simple sequencing and cause-effect, still mostly literal.",
	"B1": "B1: use a vocabulary of roughly 2,000-2,500 word families, including everyday abstract topics (opinions, plans, experiences). Use grammar such as past perfect, first and second conditionals, passive voice, reported speech. Reading passages run 130-180 words; questions test inference and the writer's main point, not just facts.",
	"B2": "B2: use a vocabulary of roughly 4,000 word families, including idiomatic and semi-formal register. Use grammar such as mixed conditionals, inversion for emphasis (Never had I..., Not only did...), cleft sentences, advanced modals of speculation. Reading passages run 180-250 words; questions test tone, implication and writer's attitude.",
	"C1": "C1: use a vocabulary of roughly 6,000-8,000 word families, including nuanced synonyms, collocations and hedging language. Use grammar such as complex subordination, discourse markers, nuanced modality, participle clauses. Reading passages run 250-320 words; questions test argument structure and subtle inference across the whole passage.",
	"C2": "C2: use a near-native vocabulary of 8,000+ word families, including idiom, register-shifting and low-frequency collocations. Use grammar with near-native command of ellipsis, complex inversion and cohesion devices with almost no restriction. Reading passages run 300-380 words; questions test the finest shades of meaning, tone and authorial intent.",
}

// LevelGuidance returns the descriptor for a CEFR level, falling back to B1
// (the documented default) for anything not in the six-level enum.
func LevelGuidance(level string) string {
	if g, ok := levelGuidance[level]; ok {
		return g
	}
	return levelGuidance["B1"]
}
