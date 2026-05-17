package services

import (
	"fmt"
	"pocka/internal/core"
)

type i18nService struct {
	dict map[string]map[string]string
}

func NewI18nService() core.I18nService {
	dict := map[string]map[string]string{
		"en": {
			"welcome_name": "👋 *Welcome to Pocka!*\n\nI'm your personal money assistant. To get started, what should I call you?\n\n_(You can type your name below)_",
			"nice_to_meet": "Nice to meet you, *%s*! 😊\n\nWhich language do you prefer?",
			"currency_prompt": "Got it! And what currency do you use most?",
			"contact_prompt": "Almost there! 🚀\n\nCan you share your phone number? This helps secure your account and link your data if you switch devices.",
			"onboarding_complete": "✨ *Onboarding Complete!*\n\nYou're all set to track your money. Just send me messages like:\n\n• `coffee 50`\n• `salary 15000`\n• `taxi 100`\n\nUse /stats anytime to see your summary. Let's grow! 🚀",
			"error_not_understood": "❌ I couldn't understand that. Try something like `100 lunch` or `salary 5000`.",
			"saved_header": "✅ *Saved:*\n",
			"nice_income": "\n_Nice! Keep it coming!_ 🚀",
			"help_text": "📖 *How to use Pocka:*\n\n1. **Log Expense**: `amount description` or `description amount` (e.g., `100 lunch` or `taxi 50`).\n2. **Log Income**: Use keywords like `salary`, `sold`, or `bonus` (e.g., `salary 15000`).\n3. **Stats**: Send /stats for a weekly breakdown of your income, expenses, and net balance.\n\nIt's that simple! No forms, no complex apps.",
			"stats_empty": "You haven't logged any transactions in the last 7 days. Start by sending something like 'coffee 40'!",
			"stats_header_weekly": "📊 *Weekly Summary (Last 7 Days)*\n\n",
			"stats_header_monthly": "📊 *Monthly Summary (This Month)*\n\n",
			"stats_income": "💰 Income: *%.2f %s*\n",
			"stats_expense": "💸 Expense: *%.2f %s*\n",
			"stats_net": "⚖️ Net: *%.2f %s*\n\n",
			"stats_top_expenses": "*Top Expenses:*\n",
			"stats_streak": "\nStreak: %d days 🔥",
			"stats_keep_tracking": "\n\n_Keep tracking to maintain your streak!_",
			"unknown_command": "I don't know that command.",
			"generating_card": "Generating your card... 🪄",
			"card_caption": "Here is your summary card! 📊 Share it with your friends to show off your financial discipline. 🚀",
			"generate_card_btn": "Generate Shareable Card 🖼️",
			"skip_btn": "Skip for now ➡️",
			"share_contact_btn": "Share Phone Number 📱",
			"share_text": "Invite your friends to Pocka and track your financial journey together!\n\nHere is your personal referral link:\n%s",
			"referral_stats": "You have invited %d friends to Pocka! 🚀",
		},
		"am": {
			"welcome_name": "👋 *እንኳን ወደ ፖካ በደህና መጡ!*\n\nእኔ የእርስዎ የግል የገንዘብ ረዳት ነኝ። ለመጀመር ምን ልበልዎ?\n\n_(ስምዎን ከታች መጻፍ ይችላሉ)_",
			"nice_to_meet": "ስለተዋወቅን ደስ ብሎኛል, *%s*! 😊\n\nየትኛውን ቋንቋ ይመርጣሉ?",
			"currency_prompt": "ገባኝ! እና የትኛውን ገንዘብ ነው በብዛት የሚጠቀሙት?",
			"contact_prompt": "ተቃርበናል! 🚀\n\nየስልክ ቁጥርዎን ማጋራት ይችላሉ? ይህ መለያዎን ደህንነቱ የተጠበቀ ለማድረግ እና መሳሪያ ቢቀይሩ ውሂብዎን ለማገናኘት ይረዳል።",
			"onboarding_complete": "✨ *ምዝገባ ተጠናቋል!*\n\nገንዘብዎን ለመከታተል ዝግጁ ነዎት። እንደዚህ ያሉ መልዕክቶችን ብቻ ይላኩልኝ፡\n\n• `ቡና 50`\n• `ደሞዝ 15000`\n• `ታክሲ 100`\n\nማጠቃለያዎን ለማየት በማንኛውም ጊዜ /stats ይጠቀሙ። 🚀",
			"error_not_understood": "❌ አልገባኝም። እንደ `100 ምሳ` ወይም `ደሞዝ 5000` ያለ ነገር ይሞክሩ።",
			"saved_header": "✅ *ተቀምጧል:*\n",
			"nice_income": "\n_ጥሩ ነው! ይቀጥሉበት!_ 🚀",
			"help_text": "📖 *ፖካን እንዴት መጠቀም እንደሚቻል:*\n\n1. **ወጪ ይመዝግቡ**: `መጠን መግለጫ` ወይም `መግለጫ መጠን` (ለምሳሌ፡ `100 ምሳ` ወይም `ታክሲ 50`).\n2. **ገቢ ይመዝግቡ**: እንደ `ደሞዝ`፣ `ተሸጠ` ወይም `ጉርሻ` ያሉ ቁልፍ ቃላትን ይጠቀሙ (ለምሳሌ፡ `ደሞዝ 15000`).\n3. **ስታትስቲክስ**: የሳምንት የገቢ፣ የወጪ እና የተጣራ ቀሪ ሂሳብዎን ለማየት /stats ይላኩ።",
			"stats_empty": "ባለፉት 7 ቀናት ውስጥ ምንም ግብይቶችን አልመዘገቡም።",
			"stats_header_weekly": "📊 *የሳምንቱ ማጠቃለያ (ባለፉት 7 ቀናት)*\n\n",
			"stats_header_monthly": "📊 *የወሩ ማጠቃለያ (በዚህ ወር)*\n\n",
			"stats_income": "💰 ገቢ: *%.2f %s*\n",
			"stats_expense": "💸 ወጪ: *%.2f %s*\n",
			"stats_net": "⚖️ የተጣራ: *%.2f %s*\n\n",
			"stats_top_expenses": "*ከፍተኛ ወጪዎች:*\n",
			"stats_streak": "\nተከታታይ ቀናት: %d ቀናት 🔥",
			"stats_keep_tracking": "\n\n_ተከታታሊነትዎን ለማስጠበቅ መከታተልዎን ይቀጥሉ!_",
			"unknown_command": "ይህን ትዕዛዝ አላውቀውም።",
			"generating_card": "ካርድዎን በማዘጋጀት ላይ... 🪄",
			"card_caption": "የማጠቃለያ ካርድዎ ይኸውልዎ! 📊 ለጓደኞችዎ ያጋሩት። 🚀",
			"generate_card_btn": "ካርድ አዘጋጅ 🖼️",
			"skip_btn": "አሁን ይለፍ ➡️",
			"share_contact_btn": "ስልክ ቁጥር አጋራ 📱",
			"share_text": "ጓደኞችዎን ወደ ፖካ ይጋብዙ እና የገንዘብ ጉዞዎን አብረው ይከታተሉ!\n\nየእርስዎ የግል የመጋበዣ ሊንክ ይኸውልዎ:\n%s",
			"referral_stats": "%d ጓደኞችን ወደ ፖካ ጋብዘዋል! 🚀",
		},
		"ti": {
			"welcome_name": "👋 *እንቋዕ ናብ ፖካ ብደሓን መጹ!*\n\nኣነ ናትኩም ናይ ገንዘብ ሓጋዚ እየ። ንምጅማር መን ክብለኩም?\n\n_(ስምኩም ኣብ ታሕቲ ክትጽሕፉ ትኽእሉ ኢኹም)_",
			"nice_to_meet": "ጽቡቕ ሌላ, *%s*! 😊\n\nኣየናይ ቋንቋ ትመርጹ?",
			"currency_prompt": "ተረዲኡኒ! ኣየናይ ዓይነት ገንዘብ ኢኹም ብብዝሒ ትጥቀሙ?",
			"contact_prompt": "ቀሪብና ኣለና! 🚀\n\nቁጽሪ ስልኪኹም ከተካፍሉና ትኽእሉዶ?",
			"onboarding_complete": "✨ *ምዝገባ ተዛዚሙ!*\n\nገንዘብኩም ንምክትታል ድሉዋት ኢኹም።\n\n• `ቡን 50`\n• `መሃያ 15000`\n• `ታክሲ 100`\n\nጸብጻብኩም ንምርኣይ /stats ተጠቐሙ። 🚀",
			"error_not_understood": "❌ ኣይተረድኣንን። ከም `100 ምሳ` ወይ `መሃያ 5000` ዝኣመሰለ ፈትኑ።",
			"saved_header": "✅ *ተዓቂቡ:*\n",
			"nice_income": "\n_ጽቡቕ ኣሎ! ቀጽልዎ!_ 🚀",
			"help_text": "📖 *ፖካ ብኸመይ ከም ዝጥቀም:*\n\n1. **ወጻኢ መዝግቡ**: `መጠን መግለጺ` ወይ `መግለጺ መጠን`\n2. **ኣታዊ መዝግቡ**: `መሃያ 15000`\n3. **ስታትስቲክስ**: /stats ስደዱ።",
			"stats_empty": "ኣብ ዝሓለፉ 7 መዓልታት ዝኾነ ትራንዛክሽን ኣይመዝገብኩምን።",
			"stats_header_weekly": "📊 *ናይ ሰሙን ጸብጻብ (ዝሓለፉ 7 መዓልታት)*\n\n",
			"stats_header_monthly": "📊 *ናይ ወርሒ ጸብጻብ (እዚ ወርሒ)*\n\n",
			"stats_income": "💰 ኣታዊ: *%.2f %s*\n",
			"stats_expense": "💸 ወጻኢ: *%.2f %s*\n",
			"stats_net": "⚖️ ዝተረፈ: *%.2f %s*\n\n",
			"stats_top_expenses": "*ለዓልቲ ወጻኢታት:*\n",
			"stats_streak": "\nተኸታታሊ መዓልታት: %d መዓልታት 🔥",
			"stats_keep_tracking": "\n\n_ተኸታታሊነትኩም ንምሕላው ምክትታልኩም ቀጽሉ!_",
			"unknown_command": "እዚ ትእዛዝ ኣይፈልጦን እየ።",
			"generating_card": "ካርድኹም ብምድላው... 🪄",
			"card_caption": "ናይ ማጠቃለያ ካርድኹም! 📊 ምስ ኣዕሩኽትኹም ተኻፈሉዎ። 🚀",
			"generate_card_btn": "ካርድ ኣዳሉ 🖼️",
			"skip_btn": "ሕጂ ሕለፍ ➡️",
			"share_contact_btn": "ስልኪ ቁጽሪ ኣካፍል 📱",
			"share_text": "ኣዕሩኽትኹም ናብ ፖካ ብምዕዳም ናይ ገንዘብ ጉዕዞኹም ብሓባር ተኸታተሉ!\n\nናትኩም ናይ መዕደሚ ሊንክ:\n%s",
			"referral_stats": "%d ኣዕሩኽትኹም ናብ ፖካ ዓዲምኩም ኣለኹም! 🚀",
		},
		"om": {
			"welcome_name": "👋 *Baga gara Pocka nagaan dhuftan!*\n\nAni gargaaraa maallaqaa keessani. Eegaluuf maal isin haa jedhu?",
			"nice_to_meet": "Wal baruun keenya gaariidha, *%s*! 😊\n\nAfaan kam filattu?",
			"currency_prompt": "Naaf galeera! Maallaqa kam baay'inaan fayyadamtu?",
			"contact_prompt": "Xumuruuf geenyeerra! 🚀\n\nLakkoofsa bilbilaa keessan qooduu dandeessuu?",
			"onboarding_complete": "✨ *Galmeen xumurameera!*\n\nMaallaqa keessan hordofuuf qophiidha.\n\n• `buna 50`\n• `mindaa 15000`\n• `taaksii 100`\n\nGabaasa keessan ilaaluuf /stats fayyadamaa. 🚀",
			"error_not_understood": "❌ Naaf hin galle. Waan akka `100 laaqana` ykn `mindaa 5000` yaadaa.",
			"saved_header": "✅ *Kaa'ameera:*\n",
			"nice_income": "\n_Baay'ee gaariidha! Itti fufaa!_ 🚀",
			"help_text": "📖 *Pocka akkamitti akka fayyadamnu:*\n\n1. **Baasii**: `100 laaqana`\n2. **Galii**: `mindaa 15000`\n3. **Istaatistiksii**: /stats ergaa.",
			"stats_empty": "Guyyoota 7 darban keessa homaa hin galmeessine.",
			"stats_header_weekly": "📊 *Gabaasa Torbanii (Guyyoota 7 darban)*\n\n",
			"stats_header_monthly": "📊 *Gabaasa Ji'aa (Ji'a Kana)*\n\n",
			"stats_income": "💰 Galii: *%.2f %s*\n",
			"stats_expense": "💸 Baasii: *%.2f %s*\n",
			"stats_net": "⚖️ Haftee: *%.2f %s*\n\n",
			"stats_top_expenses": "*Baasiiwwan gurguddoo:*\n",
			"stats_streak": "\nGuyyoota walitti aanan: Guyyaa %d 🔥",
			"stats_keep_tracking": "\n\n_Walitti fufiinsa keessan eeguuf hordofuu itti fufaa!_",
			"unknown_command": "Ajaja kana hin beeku.",
			"generating_card": "Kaardii qopheessaa jira... 🪄",
			"card_caption": "Kaardii cuunfaa keessan! 📊 Hiriyoota keessan waliin qoodaadhaa. 🚀",
			"generate_card_btn": "Kaardii Qopheessi 🖼️",
			"skip_btn": "Ammaaf dhiisi ➡️",
			"share_contact_btn": "Lakkoofsa Bilbilaa Qoodi 📱",
			"share_text": "Hiriyoota keessan gara Pocka affeeruun deemsa maallaqaa keessan waliin hordofaa!\n\nKun linkii affeerraa keessani:\n%s",
			"referral_stats": "Hiriyoota %d gara Pocka affeertaniittu! 🚀",
		},
	}
	return &i18nService{dict: dict}
}

func (s *i18nService) Translate(langCode, key string, args ...interface{}) string {
	if s.dict[langCode] == nil {
		langCode = "en" // fallback to english
	}
	val, ok := s.dict[langCode][key]
	if !ok {
		// fallback to english
		val = s.dict["en"][key]
	}
	if val == "" {
		return key
	}
	
	if len(args) > 0 {
		return fmt.Sprintf(val, args...)
	}
	return val
}
