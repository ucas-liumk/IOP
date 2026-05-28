package com.gallant.collab.service;

import com.gallant.collab.AbstractIntegrationTest;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

import static org.assertj.core.api.Assertions.assertThat;

class DashboardServiceTest extends AbstractIntegrationTest {

    @Autowired DashboardService service;

    @Test
    @DisplayName("总览 KPI 与种子数据一致")
    void overview() {
        DashboardService.DashboardData d = service.overview();
        DashboardService.Kpis k = d.getKpis();
        assertThat(k.getTotal()).isEqualTo(8);
        assertThat(k.getOverdue()).isEqualTo(1);                       // 仅 #5
        assertThat(k.getDone()).isEqualTo(1);                          // 仅 #4
        assertThat(d.getCategories()).isNotEmpty();
        assertThat(d.getTrend()).isNotEmpty();
        assertThat(d.getProcessingBreakdown()).isNotEmpty();
    }
}
