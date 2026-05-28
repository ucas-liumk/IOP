package com.gallant.collab;

import org.mybatis.spring.annotation.MapperScan;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
@MapperScan("com.gallant.collab.mapper")
public class CollabApplication {
    public static void main(String[] args) {
        SpringApplication.run(CollabApplication.class, args);
    }
}
