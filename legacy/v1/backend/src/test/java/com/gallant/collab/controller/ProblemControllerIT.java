package com.gallant.collab.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.gallant.collab.AbstractIntegrationTest;
import com.gallant.collab.dto.ProblemDtos.CreateProblemRequest;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import java.time.LocalDate;
import java.util.List;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@AutoConfigureMockMvc
class ProblemControllerIT extends AbstractIntegrationTest {

    @Autowired MockMvc mvc;
    @Autowired ObjectMapper om;

    @Test
    @DisplayName("GET /problems 返回分页 ApiResponse")
    void list() throws Exception {
        mvc.perform(get("/problems"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data.total").value(8))
                .andExpect(jsonPath("$.data.items").isArray());
    }

    @Test
    @DisplayName("GET /problems/{id} 返回详情")
    void detail() throws Exception {
        mvc.perform(get("/problems/WT20260510-001"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.problem.id").value("WT20260510-001"))
                .andExpect(jsonPath("$.data.history.length()").value(5));
    }

    @Test
    @DisplayName("POST /problems 创建问题")
    void create() throws Exception {
        CreateProblemRequest req = new CreateProblemRequest();
        req.setTitle("Mock 提交");
        req.setDescription("desc");
        req.setCategory("信息技术");
        req.setPriority("high");
        req.setDueDate(LocalDate.now().plusDays(10));
        req.setTags(List.of("test"));
        req.setParticipants(List.of("CTO 办公室"));

        mvc.perform(post("/problems")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(om.writeValueAsString(req)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.currentStage").value("review"))
                .andExpect(jsonPath("$.data.status").value("pending"));
    }

    @Test
    @DisplayName("GET /dashboard/overview 返回完整数据集")
    void dashboard() throws Exception {
        mvc.perform(get("/dashboard/overview"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.kpis.total").value(8))
                .andExpect(jsonPath("$.data.disputeStats").exists())
                .andExpect(jsonPath("$.data.trend").isArray());
    }

    @Test
    @DisplayName("GET /stages 返回 8 节点定义")
    void stages() throws Exception {
        mvc.perform(get("/stages"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.length()").value(8))
                .andExpect(jsonPath("$.data[0].code").value("submit"));
    }
}
