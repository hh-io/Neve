<template>
  <div class="tab-content">
    <section class="analytics-section fade-in">
      <div class="grid grid-2">
        <!-- Expense Pie Chart -->
        <div class="glass-card chart-card">
          <h3 class="card-title">本月支出分类</h3>
          <div class="chart-container">
            <v-chart :option="expenseChartOption" autoresize />
          </div>
        </div>
        <IncomeChart :data="analytics.incomeBreakdown" />
      </div>
    </section>

    <section class="analytics-section fade-in">
      <div class="grid grid-2">
        <PlatformRanking :data="analytics.platformRanking" />
        <MerchantRanking :data="analytics.merchantRanking" />
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import VChart from 'vue-echarts';
import IncomeChart from '../IncomeChart.vue';
import PlatformRanking from '../PlatformRanking.vue';
import MerchantRanking from '../MerchantRanking.vue';

const props = defineProps({
  analytics: { type: Object, required: true }
});

const expenseChartOption = computed(() => {
  const data = props.analytics.expenseByCategory || [];
  return {
    tooltip: {
      trigger: "item",
      formatter: "{b}: ¥{c} ({d}%)",
    },
    legend: {
      orient: "vertical",
      right: "5%",
      top: "center",
      textStyle: { color: "#6E6E73", fontSize: 12 },
    },
    series: [
      {
        type: "pie",
        radius: ["45%", "70%"],
        center: ["35%", "50%"],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: "#fff",
          borderWidth: 2,
        },
        label: { show: false },
        emphasis: {
          label: { show: true, fontSize: 14, fontWeight: "bold" },
        },
        data: data.map((item, i) => ({
          value: item.amount,
          name: item.category,
          itemStyle: {
            color: [
              "#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4",
              "#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
            ][i % 8],
          },
        })),
      },
    ],
  };
});
</script>
