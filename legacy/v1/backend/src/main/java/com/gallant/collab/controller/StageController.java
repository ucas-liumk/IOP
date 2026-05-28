package com.gallant.collab.controller;

import com.gallant.collab.stage.Stage;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/** 元数据接口: 让前端能拿到 8 阶段定义, 而不用硬编码 */
@RestController
@RequestMapping("/stages")
public class StageController {

    @GetMapping
    public List<Map<String, Object>> all() {
        return Arrays.stream(Stage.values())
                .map(s -> {
                    Map<String, Object> m = new HashMap<>();
                    m.put("code", s.code);
                    m.put("label", s.label);
                    m.put("branch", s.branch != null ? s.branch.code() : null);
                    m.put("seq", s.seq);
                    return m;
                })
                .toList();
    }
}
