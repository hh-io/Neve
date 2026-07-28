// Category utilities - shared across components
import type { Transaction } from '../types/api'

// Category label mapping (Chinese) - 全局唯一的一份映射
export const categoryLabels: Record<string, string> = {
    // 支出分类
    Food: '餐饮',
    Shopping: '购物',
    Transport: '交通',
    Entertainment: '娱乐',
    Gift: '红包/礼物',
    Financial: '金融',
    Communication: '通讯',
    Lodging: '住宿',
    Digital: '订阅',
    Health: '健康',
    Education: '教育',
    Fixed: '固定支出',
    Utilities: '公共事业',
    Housing: '居住',
    Unknown: '未分类',
    Other: '其他',
    // 收入来源
    Income: '收入',
    Salary: '工资',
    Bonus: '奖金',
    Membership: '会费',
    Dividend: '股息',
    Investment: '投资',
    SecondHand: '闲置交易',
    Family: '家人'
};

// Get localized category label
export function getCategoryLabel(category: string | undefined | null): string {
    return (category && categoryLabels[category]) || category || '其他';
}

// 展示层派生字段:processTransaction 在原始交易上叠加的视图属性。
export interface ProcessedTransaction extends Transaction {
    amount: number
    isIncome: boolean
    accountShort: string
    amountText: string
    amountClass: string
    iconClass: string
    iconColor: string
}

// Process raw transaction for display.
// 金额、分类、转账识别均由后端计算(kind/category/displayAmount/transferAmount/feeAmount),
// 这里只派生展示层字段,不再从 postings 推断业务含义。
export function processTransaction(tx: Transaction): ProcessedTransaction {
    let accountShort = '';
    for (const posting of tx.postings || []) {
        const parts = (posting.account || '').split(':');
        if (parts[0] === 'Assets' || parts[0] === 'Liabilities') {
            accountShort = parts[parts.length - 1];
            break;
        }
    }

    const kind = tx.kind || 'expense';
    const amount = tx.displayAmount ?? 0;
    const isIncome = kind === 'income' || kind === 'mixed';
    const isTransfer = kind === 'transfer';

    let amountText: string;
    let amountClass: string;
    if (isTransfer) {
        amountText = `¥${Math.abs(amount).toFixed(2)}`;
        amountClass = 'text-transfer';
    } else if (isIncome || amount < 0) {
        // 收入,或支出为负(退款)按收入方向展示
        amountText = `+¥${Math.abs(amount).toFixed(2)}`;
        amountClass = 'text-income';
    } else {
        amountText = `-¥${Math.abs(amount).toFixed(2)}`;
        amountClass = 'text-expense';
    }

    return {
        ...tx,
        amount,
        isIncome,
        isTransfer,
        category: tx.category || 'Other',
        accountShort,
        amountText,
        amountClass,
        iconClass: isTransfer ? 'bg-brand-light' : isIncome ? 'bg-income-light' : 'bg-expense-light',
        iconColor: isTransfer ? 'var(--accent)' : isIncome ? 'var(--income)' : 'var(--expense)',
        payee: tx.payee || '',
        narration: tx.narration || ''
    };
}

// 交易日期是后端按服务器时区归属的日历日,序列化成带偏移的 RFC3339
// ("2026-07-28T00:00:00+08:00")。直接 new Date() 会按**浏览器**时区重新落点,
// 浏览器偏西时整体退一天(今天的交易被标成"昨天")。故一律先截出 YYYY-MM-DD 再比,
// 只有在需要星期几时才用本地零点重建 Date——那不带任何偏移,不会漂。
function toDateKey(dateStr: string): string {
    return (dateStr || '').slice(0, 10);
}

function localDateKey(d: Date): string {
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${d.getFullYear()}-${month}-${day}`;
}

// Format date for display
export function formatTransactionDate(dateStr: string): string {
    return toDateKey(dateStr).slice(5); // "MM-DD"
}

// Get relative date label (Today, Yesterday, etc.)
export function getRelativeDateLabel(dateStr: string): string {
    const key = toDateKey(dateStr);
    if (!key) return '';

    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(today.getDate() - 1);

    if (key === localDateKey(today)) return '今天';
    if (key === localDateKey(yesterday)) return '昨天';

    const [year, month, day] = key.split('-').map(Number);
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    const weekday = weekdays[new Date(year, month - 1, day).getDay()];
    return `${month}月${day}日 ${weekday}`;
}

// Generate pastel tag color
export function getTagColor(tag: string): string {
    let hash = 0;
    for (let i = 0; i < tag.length; i++) {
        hash = tag.charCodeAt(i) + ((hash << 5) - hash);
    }
    const h = hash % 360;
    return `hsl(${h}, 30%, 90%)`;
}
