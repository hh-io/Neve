// Icon Components - Vue-native icon system
// Usage: import { IconShopping, IconFood } from '@/components/icons';
//        <IconShopping :size="20" color="var(--brand-primary)" />

// Base wrapper
export { default as IconBase } from './IconBase.vue';

// Category icons
export { default as IconShopping } from './IconShopping.vue';
export { default as IconFood } from './IconFood.vue';
export { default as IconGift } from './IconGift.vue';
export { default as IconTransfer } from './IconTransfer.vue';
export { default as IconEntertainment } from './IconEntertainment.vue';
export { default as IconHeart } from './IconHeart.vue';
export { default as IconWallet } from './IconWallet.vue';
export { default as IconCreditCard } from './IconCreditCard.vue';
export { default as IconArrowDown } from './IconArrowDown.vue';

// Icon component mapping for categories
import IconShopping from './IconShopping.vue';
import IconFood from './IconFood.vue';
import IconGift from './IconGift.vue';
import IconTransfer from './IconTransfer.vue';
import IconEntertainment from './IconEntertainment.vue';
import IconHeart from './IconHeart.vue';
import IconWallet from './IconWallet.vue';
import IconCreditCard from './IconCreditCard.vue';
import IconArrowDown from './IconArrowDown.vue';

export const categoryIconComponents = {
    Food: IconFood,
    Shopping: IconShopping,
    Transport: IconTransfer,
    Gift: IconGift,
    Entertainment: IconEntertainment,
    Financial: IconWallet,
    Income: IconArrowDown,
    Health: IconHeart,
    Digital: IconCreditCard,
    Education: IconCreditCard,
    Communication: IconCreditCard,
    Lodging: IconCreditCard,
    Other: IconCreditCard
};

// Get icon component by category
export function getCategoryIconComponent(category) {
    return categoryIconComponents[category] || IconCreditCard;
}
