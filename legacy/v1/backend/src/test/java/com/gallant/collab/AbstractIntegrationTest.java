package com.gallant.collab;

import com.gallant.collab.common.UserContext;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.io.ClassPathResource;
import org.springframework.jdbc.datasource.init.ScriptUtils;
import org.springframework.test.context.ActiveProfiles;

import javax.sql.DataSource;
import java.sql.Connection;

/**
 * 集成测试基类。
 * 依赖本地运行的 PostgreSQL (docker compose up -d db)。
 * 每个测试类执行前重置 schema + 灌入种子, 保证测试间隔离。
 */
@SpringBootTest
@ActiveProfiles("test")
public abstract class AbstractIntegrationTest {

    @Autowired DataSource dataSource;

    /** 每个测试方法前重置 schema + seed, 保证测试间完全隔离 */
    @BeforeEach
    void initSchema() throws Exception {
        try (Connection c = dataSource.getConnection()) {
            ScriptUtils.executeSqlScript(c, new ClassPathResource("schema.sql"));
            ScriptUtils.executeSqlScript(c, new ClassPathResource("seed.sql"));
        }
        UserContext.set(new UserContext.CurrentUser(1L, "陈雨晴", "CTO 办公室"));
    }

    @AfterEach
    void clearUser() {
        UserContext.clear();
    }
}
