package com.gallant.collab.controller;

import com.gallant.collab.domain.Attachment;
import com.gallant.collab.service.FileStorageService;
import lombok.RequiredArgsConstructor;
import org.springframework.core.io.InputStreamResource;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

import java.io.InputStream;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.util.List;

@RestController
@RequestMapping("/files")
@RequiredArgsConstructor
public class FileController {

    private final FileStorageService service;

    @PostMapping("/upload")
    public Attachment upload(@RequestParam String problemId,
                             @RequestParam String stage,
                             @RequestParam MultipartFile file) {
        return service.upload(problemId, stage, file);
    }

    @GetMapping("/problem/{problemId}")
    public List<Attachment> listByProblem(@PathVariable String problemId) {
        return service.listByProblem(problemId);
    }

    @GetMapping("/{id}/download")
    public ResponseEntity<InputStreamResource> download(@PathVariable Long id) {
        Attachment meta = service.requireMeta(id);
        InputStream in = service.download(id);
        String fn = URLEncoder.encode(meta.getFileName(), StandardCharsets.UTF_8);
        return ResponseEntity.ok()
                .header(HttpHeaders.CONTENT_DISPOSITION, "attachment; filename*=UTF-8''" + fn)
                .contentType(MediaType.parseMediaType(
                        meta.getContentType() != null ? meta.getContentType() : MediaType.APPLICATION_OCTET_STREAM_VALUE))
                .contentLength(meta.getFileSize() != null ? meta.getFileSize() : -1)
                .body(new InputStreamResource(in));
    }

    @DeleteMapping("/{id}")
    public void delete(@PathVariable Long id) {
        service.delete(id);
    }
}
