package main

// facts are what the reminder says once ninety minutes are gone. Each one is
// true as stated, with the study behind it in the comment: a scare that turns
// out to be made up stops working the day you look it up.
var facts = []string{
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
