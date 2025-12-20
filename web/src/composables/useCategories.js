// Category utilities - shared across components
import { icons } from './icons';

// Category label mapping (Chinese)
export const categoryLabels = {
    Food: '餐饮',
    Shopping: '购物',
    Transport: '交通',
    Entertainment: '娱乐',
    Gift: '红包/礼物',
    Financial: '金融',
    Communication: '通讯',
    Lodging: '住宿',
    Digital: '数码',
    Health: '健康',
    Education: '教育',
    Income: '收入',
    Other: '其他'
};

// Get localized category label
export function getCategoryLabel(category) {
    return categoryLabels[category] || category || '其他';
}

// Category to icon mapping
export function getCategoryIcon(category) {
    const iconMap = {
        Food: icons.food,
        Shopping: icons.shopping,
        Transport: icons.transfer,
        Gift: icons.gift,
        Entertainment: icons.entertainment,
        Financial: icons.wallet,
        Income: icons.arrowDown,
        Health: icons.heart,
        Digital: icons.creditCard,
        Education: icons.tags,
        Communication: icons.bell,
        Lodging: icons.bank,
        Other: icons.creditCard
    };
    return iconMap[category] || icons.creditCard;
}

// Process raw transaction to extract amount, category, etc.
export function processTransaction(tx) {
    // If already processed, return as-is
    if (tx.isIncome !== undefined && tx.amount !== undefined) {
        return tx;
    }

    let amount = 0;
    let category = 'Other';
    let isIncome = false;
    let accountShort = '';
    let isTransfer = false;

    if (tx.postings && tx.postings.length > 0) {
        for (const posting of tx.postings) {
            const account = posting.account || '';

            if (account.startsWith('Expenses:')) {
                amount = posting.amount;
                isIncome = false;
                const parts = account.split(':');
                category = parts.length > 1 ? parts[1] : 'Other';
            } else if (account.startsWith('Income:')) {
                amount = Math.abs(posting.amount);
                isIncome = true;
                const parts = account.split(':');
                category = parts.length > 1 ? parts[1] : 'Income';
            }

            if (account.startsWith('Assets:') || account.startsWith('Liabilities:')) {
                const parts = account.split(':');
                accountShort = parts.length > 2 ? parts[2] : (parts.length > 1 ? parts[1] : account);
            }
        }

        // 内部转账处理：仅涉及 Assets/Liabilities 的交易（如还款、转账）
        if (amount === 0 && tx.postings.length >= 2) {
            const hasExpenseOrIncome = tx.postings.some(p => 
                p.account?.startsWith('Expenses:') || p.account?.startsWith('Income:')
            );
            
            if (!hasExpenseOrIncome) {
                isTransfer = true;
                category = 'Financial';
                // 取正值金额作为转账金额
                for (const posting of tx.postings) {
                    if (posting.amount > 0) {
                        amount = posting.amount;
                        // Liabilities 正值表示还款
                        if (posting.account?.startsWith('Liabilities:')) {
                            isIncome = false; // 还款视为支出方向
                        }
                        break;
                    }
                }
            }
        }
    }

    return {
        ...tx,
        amount,
        category,
        isIncome,
        isTransfer,
        accountShort,
        payee: tx.payee || '',
        narration: tx.narration || ''
    };
}

// Format date for display
export function formatTransactionDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${month}-${day}`;
}

// Get relative date label (Today, Yesterday, etc.)
export function getRelativeDateLabel(dateStr) {
    const date = new Date(dateStr);
    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(today.getDate() - 1);

    date.setHours(0, 0, 0, 0);
    today.setHours(0, 0, 0, 0);
    yesterday.setHours(0, 0, 0, 0);

    if (date.getTime() === today.getTime()) {
        return '今天';
    } else if (date.getTime() === yesterday.getTime()) {
        return '昨天';
    } else {
        const month = date.getMonth() + 1;
        const day = date.getDate();
        const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
        const weekday = weekdays[date.getDay()];
        return `${month}月${day}日 ${weekday}`;
    }
}

// Generate pastel tag color
export function getTagColor(tag) {
    let hash = 0;
    for (let i = 0; i < tag.length; i++) {
        hash = tag.charCodeAt(i) + ((hash << 5) - hash);
    }
    const h = hash % 360;
    return `hsl(${h}, 30%, 90%)`;
}
