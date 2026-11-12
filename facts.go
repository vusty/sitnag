//go:build windows

package main

// The facts are what the reminder says once ninety minutes are gone. Each one
// is true as stated and names its study on the card, with the figures behind
// it in the comment: a scare that turns out to be made up stops working the
// day you look it up.
//
// They are about what sitting does and what getting up undoes, not about a
// daily total: "eight hours a day" is easy to wave away as someone else's
// problem. Two popular lines are left out on purpose. "Sitting is the new
// smoking" does not hold: Vallance et al., Am J Public Health 2018, put the
// excess deaths from smoking at ten times those from sitting. And no study
// compares sitting with alcohol head to head.
type fact struct {
	source string // shown on the card, the same in both languages
	ru, en string
}

var facts = []fact{
	// Femoral artery flow-mediated dilation fell from 4.72% to 2.2% over three
	// hours of sitting; with a 5-minute walk each hour it did not fall. A
	// meta-analysis of 22 trials agrees: Paterson et al., Sports Med 2020.
	{"Thosar et al., Med Sci Sports Exerc, 2015",
		"Пока ты сидишь неподвижно, артерии ног вдвое хуже расширяются в ответ на кровоток. Пять минут ходьбы раз в час это полностью предотвращают.",
		"While you sit still, the leg arteries lose half of their ability to widen with blood flow. A five-minute walk every hour prevents it completely."},
	// "eThrombosis": a 32-year-old man, life-threatening embolism after long
	// periods at a computer. Case-control data: Healy et al., J R Soc Med 2010.
	{"Beasley et al., Eur Respir J, 2003",
		"Икры перестают качать кровь к сердцу, и она застаивается в венах голеней. Так образуются тромбы: тромбоэмболия после долгого сидения за компьютером описана в медицинских журналах.",
		"Idle calves stop pumping blood back to the heart, and it pools in the veins of the lower legs. That is how clots form: embolism after long hours at a computer is in the medical journals."},
	// Middle cerebral artery velocity fell over four hours of sitting and did
	// not with 2-minute walks every 30 minutes.
	{"Carter et al., J Appl Physiol, 2018",
		"Сидение без перерывов снижает кровоток в мозге. Две минуты ходьбы каждые полчаса не дают ему упасть.",
		"Sitting without a break cuts blood flow in the brain. Two minutes of walking every half hour keep it from falling."},
	// 13 studies, 1 005 791 people: most sitting and least active against
	// least sitting and most active, hazard ratio 1.59.
	{"Ekelund et al., Lancet, 2016",
		"У тех, кто много сидит и мало двигается, риск ранней смерти на 59% выше, чем у тех, кто сидит мало и двигается много. Это данные о миллионе человек.",
		"People who sit a lot and move little are 59% more likely to die early than those who sit little and move a lot. That is from data on a million people."},
	// Meta-analysis: sedentary time against type 2 diabetes (HR 1.91),
	// cardiovascular disease and cancer, adjusted for physical activity.
	{"Biswas et al., Ann Intern Med, 2015",
		"Чем больше человек сидит, тем чаще у него диабет 2 типа, болезни сердца и рак. Связь остаётся и с поправкой на то, сколько он занимается спортом.",
		"The more a person sits, the more often they get type 2 diabetes, heart disease and cancer. The link holds after accounting for how much they exercise."},
	// Highest against lowest sedentary time: diabetes RR 2.12, cardiovascular
	// mortality HR 1.90.
	{"Wilmot et al., Diabetologia, 2012",
		"У тех, кто сидит больше всех, диабет 2 типа встречается вдвое чаще, чем у тех, кто сидит меньше всех, а смерть от болезней сердца на 90% чаще.",
		"Those who sit the most get type 2 diabetes twice as often as those who sit the least, and die of heart disease 90% more often."},
	// A life-table model: 21.8 minutes per hour of television. The cigarette
	// is Shaw et al., BMJ 2000; the journal's release sets the two side by side.
	{"Veerman et al., Br J Sports Med, 2012",
		"Каждый час перед телевизором после 25 лет отнимает в среднем 22 минуты жизни. Одна сигарета отнимает 11.",
		"Every hour in front of the TV after the age of 25 takes an average of 22 minutes off a life. One cigarette takes 11."},
	// 7985 adults wearing accelerometers: longest against shortest mean
	// sedentary bout, hazard ratio 1.96.
	{"Diaz et al., Ann Intern Med, 2017",
		"Важно не только сколько сидишь, но и сколько подряд. У людей с самыми долгими отрезками сидения без перерыва риск смерти почти вдвое выше, чем с самыми короткими.",
		"It is not only how much you sit but how long at a stretch. People with the longest unbroken bouts of sitting are almost twice as likely to die as those with the shortest."},
	// Meta-analysis of 24 studies: highest against lowest, RR 1.25.
	{"Zhai et al., Br J Sports Med, 2015",
		"У тех, кто много сидит, депрессия встречается на 25% чаще, чем у тех, кто сидит мало.",
		"Depression is 25% more common in those who sit a lot than in those who sit little."},
	// Highest against lowest: colon 1.24 and endometrial 1.32 for total
	// sitting, lung 1.21 for sedentary behaviour overall.
	{"Schmid & Leitzmann, J Natl Cancer Inst, 2014",
		"У тех, кто сидит больше всех, рак толстой кишки встречается на 24% чаще, рак эндометрия на 32%, рак лёгких на 21%.",
		"Those who sit the most get colon cancer 24% more often, endometrial cancer 32% more often and lung cancer 21% more often."},
	// Breaks in sedentary time against waist, BMI, triglycerides and 2-hour
	// glucose, independent of total sedentary time.
	{"Healy et al., Diabetes Care, 2008",
		"Чем чаще человек встаёт, тем меньше у него талия и ниже сахар и жиры в крови. Это так даже при одинаковом общем времени сидения.",
		"The more often a person gets up, the smaller their waist and the lower their blood sugar and blood fats. That holds even for the same total time spent sitting."},
	// Glucose after a test drink 24% lower with light 2-minute walks.
	{"Dunstan et al., Diabetes Care, 2012",
		"Сидение держит сахар в крови высоким. Две минуты ходьбы каждые 20 минут снижают его подъём после еды примерно на четверть.",
		"Sitting keeps blood sugar high. Two minutes of walking every 20 minutes cut its rise after a meal by about a quarter."},
	// 70 adults, nine hours: walks of 1 min 40 s every 30 minutes against one
	// 30-minute walk.
	{"Peddie et al., Am J Clin Nutr, 2013",
		"Короткие проходки каждые полчаса снижают сахар в крови после еды сильнее, чем одна получасовая прогулка за день.",
		"Short walks every half hour lower blood sugar after meals more than a single half-hour walk does."},
	// Lipoprotein lipase in the hindlimb muscle of rats kept from walking.
	{"Bey & Hamilton, J Physiol, 2003",
		"В неподвижных мышцах ног почти выключается фермент, который забирает жир из крови: в опытах на крысах его активность падала на 90%.",
		"In still leg muscles the enzyme that clears fat from the blood all but switches off: in rats its activity dropped by 90%."},
	// 3-minute light walks every 30 minutes; fatigue lower at four and at
	// seven hours.
	{"Wennberg et al., BMJ Open, 2016",
		"Три минуты ходьбы каждые полчаса заметно снижают усталость к концу рабочего дня по сравнению с сидением без перерывов.",
		"Three minutes of walking every half hour leave people noticeably less tired by the end of the day than sitting without a break."},
	// Popliteal artery dilation over three hours of sitting: 4.5% to 1.6% in
	// the still leg, 3.7% to 6.6% in the leg that fidgeted.
	{"Morishima et al., Am J Physiol, 2016",
		"В опыте у сидящих людей артерия неподвижной ноги стала расширяться втрое хуже. В ноге, которой шевелили минуту из каждых пяти, она работала лучше, чем до опыта.",
		"In people kept sitting, the artery of the leg held still lost two thirds of its ability to widen. In the leg they moved for one minute in five it worked better than before."},
	// Systolic pressure 5.2 mmHg lower with one minute of walking every hour.
	{"Duran et al., Med Sci Sports Exerc, 2023",
		"Даже минута ходьбы раз в час снижает давление: в опыте на 5 мм рт. ст. по сравнению с сидением без перерывов.",
		"Even a minute of walking once an hour lowers blood pressure: by 5 mmHg in one trial, compared with sitting without a break."},
	// "Adults should limit the amount of time spent being sedentary. Replacing
	// sedentary time with physical activity of any intensity (including light
	// intensity) provides health benefits."
	{"WHO guidelines, 2020",
		"ВОЗ с 2020 года отдельно советует взрослым меньше сидеть: замена сидения движением любой интенсивности, даже лёгким, идёт на пользу здоровью.",
		"Since 2020 the WHO has advised adults to limit the time they spend sitting: replacing it with movement of any intensity, even light, benefits health."},
}

// text is the fact in the language in use.
func (f fact) text() string {
	if t.lang == "ru" {
		return f.ru
	}
	return f.en
}
