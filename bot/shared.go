package bot

var Msg = struct {
	AlertStart                  string
	AlertListApp                string
	AlertTrialAccess            string
	AlertSubscriptions          string
	AlertSetupInstruction       string
	AlertPostImportInstructions string
	AlertSuccess                string
}{
	AlertStart: `
🔥 <b>NeonGuard</b> — твой интернет без ограничений

📡 Мгновенное подключение. Без логов. Без следов.

🛫 Доступ к зарубежным сайтам и приложениям.

🚀 Настройка занимает меньше минуты — всё готово.

<b>NeonGuard</b> - Это быстрый, безопасный и простой способ подключиться к миру.
`,

	AlertTrialAccess: `
<b>Ваша подписка активна до 07.06.2025</b>

Ваша подписка: 3 дня

Вы можете пользоваться ВПН на ваших устройствах, без ограничений.
`,

	AlertListApp: `
	Ниже вы можете скачать наше приложение для iOS и MacOS.
`,
	AlertSubscriptions: `
<b>Выберите срок подписки.</b>

Чем больше срок подписки, тем ниже стоимость одного месяца.
`,
	AlertSetupInstruction: `
Если скачали приложение, нажмите Автонастройка и конфиг автоматически подтянеться.
`,

	AlertPostImportInstructions: `
Все готово, идите включайтесь

Если не сработало жмите на ручную настройку
`,

	AlertSuccess: `
Круто, если возникунт проблемы пишите нам

Подписка действует до 07.06.2025
`,
}

const (
	ErrDefault = "Что-то пошло не так. Мы уже работаем над этим!"
)
