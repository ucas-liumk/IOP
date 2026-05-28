package com.gallant.collab.controller;

import com.gallant.collab.domain.Message;
import com.gallant.collab.service.MessageService;
import jakarta.validation.constraints.NotBlank;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/messages")
@RequiredArgsConstructor
public class MessageController {

    private final MessageService service;

    @GetMapping("/problem/{problemId}")
    public List<Message> list(@PathVariable String problemId) {
        return service.list(problemId);
    }

    @PostMapping("/problem/{problemId}")
    public Message post(@PathVariable String problemId, @RequestBody PostRequest req) {
        return service.post(problemId, req.getContent());
    }

    @Data
    public static class PostRequest {
        @NotBlank private String content;
    }
}
