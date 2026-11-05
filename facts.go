//go:build windows

package main

// The facts are what the reminder says once ninety minutes are gone. Each one
// is true as stated, with the study behind it in the comment: a scare that
// turns out to be made up stops working the day you look it up. The two lists
// are the same facts in the same order.
var factsRU = []string{
	// Thosar et al., Med Sci Sports Exerc 2015.
	"Три часа сидения вдвое снижают способность артерий ног расширяться. Пять минут ходьбы каждый час это полностью предотвращают.",
	// Calf muscle pump; DVT case reports after long gaming sessions.
	"Икры не качают кровь обратно к сердцу, и она застаивается в венах голеней. Так образуются тромбы: известны случаи тромбоэмболии у геймеров после многочасовых сессий.",
	// Carter et al., J Appl Physiol 2018.
	"Четыре часа сидения без перерыва заметно снижают приток крови к мозгу. Две минуты ходьбы каждые полчаса это убирают.",
	// Ekelund et al., Lancet 2016.
	"Больше восьми часов сидения в день без движения повышают риск ранней смерти на 59%, это уровень курения и ожирения. Час активности в день этот риск снимает.",
	// Wall et al., Acta Physiol 2014: five days of leg immobilisation.
	"Мышца без нагрузки атрофируется быстро: за 5 дней неподвижности бедро теряет около 4% массы и 9% силы. За компом мышцы ног почти не работают.",
	// Disc nutrition by diffusion driven by load changes.
	"Межпозвонковые диски не имеют своих сосудов: питание в них попадает только при движении и смене нагрузки. В одной позе они голодают и теряют воду.",
	// Dunstan et al., Diabetes Care 2012.
	"Сидение держит сахар в крови высоким. Две минуты ходьбы каждые 20 минут снижают его пик после еды примерно на четверть.",
	// Hamilton et al., lipoprotein lipase in inactive muscle.
	"В неподвижных мышцах ног почти выключается фермент, который забирает жир из крови, и жир дольше остаётся в сосудах.",
	// Hip flexor shortening and lumbar load in prolonged sitting.
	"Сгибатели бедра укорачиваются, ягодицы и пресс выключаются, и спину начинает держать поясница. Отсюда боль в пояснице у всех, кто сидит часами.",
}

var factsEN = []string{
	// Thosar et al., Med Sci Sports Exerc 2015.
	"Three hours of sitting halve the ability of the leg arteries to widen. A five-minute walk every hour prevents it completely.",
	// Calf muscle pump; DVT case reports after long gaming sessions.
	"Idle calves stop pumping blood back to the heart, and it pools in the veins of the lower legs. That is how clots form: gamers have had pulmonary embolisms after marathon sessions.",
	// Carter et al., J Appl Physiol 2018.
	"Four hours of uninterrupted sitting noticeably cut blood flow to the brain. Two minutes of walking every half hour undo it.",
	// Ekelund et al., Lancet 2016.
	"More than eight hours of sitting a day without exercise raise the risk of early death by 59%, on a par with smoking and obesity. An hour of activity a day removes that risk.",
	// Wall et al., Acta Physiol 2014: five days of leg immobilisation.
	"Unused muscle wastes fast: five days without moving cost the thigh about 4% of its mass and 9% of its strength. At the computer the leg muscles barely work.",
	// Disc nutrition by diffusion driven by load changes.
	"Spinal discs have no blood vessels of their own: nutrients only get in when you move and the load changes. Held in one posture they starve and lose water.",
	// Dunstan et al., Diabetes Care 2012.
	"Sitting keeps blood sugar high. Two minutes of walking every 20 minutes cut its peak after a meal by about a quarter.",
	// Hamilton et al., lipoprotein lipase in inactive muscle.
	"In still leg muscles the enzyme that clears fat from the blood nearly switches off, and the fat stays in the vessels longer.",
	// Hip flexor shortening and lumbar load in prolonged sitting.
	"The hip flexors shorten, the glutes and abs switch off, and the lower back ends up holding you up. Hence the back pain of everyone who sits for hours.",
}
