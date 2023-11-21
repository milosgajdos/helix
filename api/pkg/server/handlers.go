package server

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/lukemarsden/helix/api/pkg/controller"
	"github.com/lukemarsden/helix/api/pkg/dataprep/text"
	"github.com/lukemarsden/helix/api/pkg/filestore"
	"github.com/lukemarsden/helix/api/pkg/model"
	"github.com/lukemarsden/helix/api/pkg/store"
	"github.com/lukemarsden/helix/api/pkg/system"
	"github.com/lukemarsden/helix/api/pkg/types"
	"github.com/rs/zerolog/log"
)

func (apiServer *HelixAPIServer) config(res http.ResponseWriter, req *http.Request) (types.ServerConfig, error) {
	if apiServer.Options.LocalFilestorePath != "" {
		return types.ServerConfig{
			FilestorePrefix: fmt.Sprintf("%s%s/filestore/viewer", apiServer.Options.URL, API_PREFIX),
		}, nil
	}

	// TODO: work out what to do for object storage here
	return types.ServerConfig{}, fmt.Errorf("we currently only support local filestore")
}

func (apiServer *HelixAPIServer) status(res http.ResponseWriter, req *http.Request) (types.UserStatus, error) {
	return apiServer.Controller.GetStatus(apiServer.getRequestContext(req))
}

func (apiServer *HelixAPIServer) getTransactions(res http.ResponseWriter, req *http.Request) ([]*types.BalanceTransfer, error) {
	return apiServer.Controller.GetTransactions(apiServer.getRequestContext(req))
}

func (apiServer *HelixAPIServer) filestoreConfig(res http.ResponseWriter, req *http.Request) (filestore.FilestoreConfig, error) {
	return apiServer.Controller.FilestoreConfig(apiServer.getRequestContext(req))
}

func (apiServer *HelixAPIServer) filestoreList(res http.ResponseWriter, req *http.Request) ([]filestore.FileStoreItem, error) {
	return apiServer.Controller.FilestoreList(apiServer.getRequestContext(req), req.URL.Query().Get("path"))
}

func (apiServer *HelixAPIServer) filestoreGet(res http.ResponseWriter, req *http.Request) (filestore.FileStoreItem, error) {
	return apiServer.Controller.FilestoreGet(apiServer.getRequestContext(req), req.URL.Query().Get("path"))
}

func (apiServer *HelixAPIServer) filestoreCreateFolder(res http.ResponseWriter, req *http.Request) (filestore.FileStoreItem, error) {
	return apiServer.Controller.FilestoreCreateFolder(apiServer.getRequestContext(req), req.URL.Query().Get("path"))
}

func (apiServer *HelixAPIServer) filestoreRename(res http.ResponseWriter, req *http.Request) (filestore.FileStoreItem, error) {
	return apiServer.Controller.FilestoreRename(apiServer.getRequestContext(req), req.URL.Query().Get("path"), req.URL.Query().Get("new_path"))
}

func (apiServer *HelixAPIServer) filestoreDelete(res http.ResponseWriter, req *http.Request) (string, error) {
	path := req.URL.Query().Get("path")
	err := apiServer.Controller.FilestoreDelete(apiServer.getRequestContext(req), path)
	return path, err
}

// TODO version of this which is session specific
func (apiServer *HelixAPIServer) filestoreUpload(res http.ResponseWriter, req *http.Request) (bool, error) {
	path := req.URL.Query().Get("path")
	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		return false, err
	}

	files := req.MultipartForm.File["files"]
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return false, fmt.Errorf("unable to open file")
		}
		defer file.Close()
		_, err = apiServer.Controller.FilestoreUpload(apiServer.getRequestContext(req), filepath.Join(path, fileHeader.Filename), file)
		if err != nil {
			return false, fmt.Errorf("unable to upload file: %s", err.Error())
		}
	}

	return true, nil
}

func (apiServer *HelixAPIServer) convertFilestorePath(ctx context.Context, sessionID string, filePath string) (string, types.RequestContext, error) {
	session, err := apiServer.Store.GetSession(ctx, sessionID)
	if err != nil {
		return "", types.RequestContext{}, err
	}

	if session == nil {
		return "", types.RequestContext{}, fmt.Errorf("no session found with id %v", sessionID)
	}

	requestContext := types.RequestContext{
		Ctx:       ctx,
		Owner:     session.Owner,
		OwnerType: session.OwnerType,
	}
	// let's remove the /dev/users/XXX part of the path if it's there
	userPath, err := apiServer.Controller.GetFilestoreUserPath(requestContext, "")
	if err != nil {
		return "", types.RequestContext{}, err
	}

	if strings.HasPrefix(filePath, userPath) {
		filePath = strings.TrimPrefix(filePath, userPath)
	}

	return filePath, requestContext, nil
}

// in this case the path contains the full /dev/users/XXX/sessions/XXX path
// so we need to remove the /dev/users/XXX part and then we load the session based on it's ID
func (apiServer *HelixAPIServer) runnerSessionDownloadFile(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionid := vars["sessionid"]
	filePath := req.URL.Query().Get("path")
	filename := filepath.Base(filePath)

	log.Debug().
		Msgf("🔵 download file: %s", filePath)

	err := func() error {
		filePath, requestContext, err := apiServer.convertFilestorePath(req.Context(), sessionid, filePath)
		if err != nil {
			return err
		}
		stream, err := apiServer.Controller.FilestoreDownload(requestContext, filePath)
		if err != nil {
			return err
		}

		// Set the appropriate mime-type headers
		res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		res.Header().Set("Content-Type", http.DetectContentType([]byte(filename)))

		// Write the file to the http.ResponseWriter
		_, err = io.Copy(res, stream)
		if err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		log.Error().Msgf("error for download file: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (apiServer *HelixAPIServer) runnerSessionDownloadFolder(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionid := vars["sessionid"]
	filePath := req.URL.Query().Get("path")
	filename := filepath.Base(filePath)

	log.Debug().
		Msgf("🔵 download folder: %s", filePath)

	err := func() error {
		filePath, requestContext, err := apiServer.convertFilestorePath(req.Context(), sessionid, filePath)
		if err != nil {
			return err
		}
		tarStream, err := apiServer.Controller.FilestoreDownloadFolder(requestContext, filePath)
		if err != nil {
			return err
		}

		// Set the appropriate mime-type headers
		res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s,tar", filename))
		res.Header().Set("Content-Type", "application/x-tar")

		// Write the file to the http.ResponseWriter
		_, err = io.Copy(res, tarStream)
		if err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		log.Error().Msgf("error for download file: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

// TODO: this need auth because right now it's an open filestore
func (apiServer *HelixAPIServer) runnerSessionUploadFiles(res http.ResponseWriter, req *http.Request) ([]filestore.FileStoreItem, error) {
	vars := mux.Vars(req)
	sessionid := vars["sessionid"]
	filePath := req.URL.Query().Get("path")

	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		return nil, err
	}

	session, err := apiServer.Store.GetSession(req.Context(), sessionid)
	if err != nil {
		return nil, err
	}

	uploadFolder := filepath.Join(controller.GetSessionFolder(session.ID), filePath)

	reqContext := types.RequestContext{
		Ctx:       req.Context(),
		Owner:     session.Owner,
		OwnerType: session.OwnerType,
	}

	result := []filestore.FileStoreItem{}
	files := req.MultipartForm.File["files"]

	for _, fileHeader := range files {
		// Handle non-tar files as before
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("unable to open file")
		}
		defer file.Close()

		item, err := apiServer.Controller.FilestoreUpload(reqContext, filepath.Join(uploadFolder, fileHeader.Filename), file)
		if err != nil {
			return nil, fmt.Errorf("unable to upload file: %s", err.Error())
		}
		result = append(result, item)
	}

	return result, nil
}

func (apiServer *HelixAPIServer) runnerSessionUploadFolder(res http.ResponseWriter, req *http.Request) (*filestore.FileStoreItem, error) {
	vars := mux.Vars(req)
	sessionid := vars["sessionid"]
	filePath := req.URL.Query().Get("path")

	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		return nil, err
	}

	session, err := apiServer.Store.GetSession(req.Context(), sessionid)
	if err != nil {
		return nil, err
	}

	uploadFolder := filepath.Join(controller.GetSessionFolder(session.ID), filePath)

	reqContext := types.RequestContext{
		Ctx:       req.Context(),
		Owner:     session.Owner,
		OwnerType: session.OwnerType,
	}

	files := req.MultipartForm.File["files"]

	if len(files) != 1 {
		return nil, fmt.Errorf("upload folder only supports a single file")
	}

	file := files[0]

	if !strings.HasSuffix(file.Filename, ".tar") {
		return nil, fmt.Errorf("upload folder only supports a tar file")
	}

	log.Debug().Msgf("🟠 Got tar file %s", file.Filename)

	// Handle .tar file
	tarFile, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("unable to open tar file")
	}
	defer func() {
		tarFile.Close()
	}()

	tarReader := tar.NewReader(tarFile)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar file: %s", err)
		}
		if header.Typeflag == tar.TypeReg {
			buffer := bytes.NewBuffer(nil)
			if _, err := io.Copy(buffer, tarReader); err != nil {
				return nil, fmt.Errorf("error reading file inside tar: %s", err)
			}

			// Create a virtual file from the buffer to upload
			vFile := bytes.NewReader(buffer.Bytes())
			_, err := apiServer.Controller.FilestoreUpload(reqContext, filepath.Join(uploadFolder, header.Name), vFile)
			if err != nil {
				return nil, fmt.Errorf("unable to upload file: %s", err.Error())
			}
		}
	}

	finalFolder, err := apiServer.Controller.FilestoreGet(reqContext, uploadFolder)
	if err != nil {
		return nil, err
	}

	return &finalFolder, nil
}

func (apiServer *HelixAPIServer) getSession(res http.ResponseWriter, req *http.Request) (*types.Session, error) {
	vars := mux.Vars(req)
	id := vars["id"]

	reqContext := apiServer.getRequestContext(req)
	session, err := apiServer.Store.GetSession(reqContext.Ctx, id)
	if err != nil {
		return nil, err
	}
	if session.OwnerType != reqContext.OwnerType || session.Owner != reqContext.Owner {
		return nil, fmt.Errorf("access denied")
	}
	return session, nil
}

func (apiServer *HelixAPIServer) getSessions(res http.ResponseWriter, req *http.Request) ([]*types.Session, error) {
	reqContext := apiServer.getRequestContext(req)
	query := store.GetSessionsQuery{}
	query.Owner = reqContext.Owner
	query.OwnerType = reqContext.OwnerType
	return apiServer.Store.GetSessions(reqContext.Ctx, query)
}

func (apiServer *HelixAPIServer) getSessionFinetuneConversation(res http.ResponseWriter, req *http.Request) ([]text.ShareGPTConversations, error) {
	vars := mux.Vars(req)
	id := vars["id"]
	reqContext := apiServer.getRequestContext(req)

	session, err := apiServer.Store.GetSession(reqContext.Ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("no session found with id %v", id)
	}
	if session.OwnerType != reqContext.OwnerType || session.Owner != reqContext.Owner {
		return nil, fmt.Errorf("access denied")
	}

	systemInteraction, err := model.GetSystemInteraction(session)
	if err != nil {
		return nil, err
	}
	if len(systemInteraction.Files) == 0 {
		return nil, fmt.Errorf("no files found")
	}
	filepath := systemInteraction.Files[0]
	if !strings.HasSuffix(filepath, ".jsonl") {
		return nil, fmt.Errorf("file is not a jsonl file")
	}
	return apiServer.Controller.ReadTextFineTuneQuestions(filepath)
}

func (apiServer *HelixAPIServer) setSessionFinetuneConversation(res http.ResponseWriter, req *http.Request) ([]text.ShareGPTConversations, error) {
	vars := mux.Vars(req)
	id := vars["id"]
	reqContext := apiServer.getRequestContext(req)

	session, err := apiServer.Store.GetSession(reqContext.Ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("no session found with id %v", id)
	}
	if session.OwnerType != reqContext.OwnerType || session.Owner != reqContext.Owner {
		return nil, fmt.Errorf("access denied")
	}

	systemInteraction, err := model.GetSystemInteraction(session)
	if err != nil {
		return nil, err
	}
	if len(systemInteraction.Files) == 0 {
		return nil, fmt.Errorf("no files found")
	}
	filepath := systemInteraction.Files[0]
	if !strings.HasSuffix(filepath, ".jsonl") {
		return nil, fmt.Errorf("file is not a jsonl file")
	}

	var data []text.ShareGPTConversations

	// Decode the JSON from the request body
	err = json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	err = apiServer.Controller.WriteTextFineTuneQuestions(filepath, data)
	if err != nil {
		return nil, err
	}

	// now we switch the session into training mode
	err = apiServer.Controller.BeginTextFineTune(session)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// based on a multi-part form that has message and files
// return a user interaction we can add to a session
// if we are uploading files for a fine-tuning session for images
// then the form data will have a field named after each of the filenames
// this is the label for the file and we should create a text file
// in the session folder that is named after the file and contains the label
func (apiServer *HelixAPIServer) getUserInteractionFromForm(
	req *http.Request,
	sessionID string,
	sessionMode types.SessionMode,
) (*types.Interaction, error) {
	message := req.FormValue("input")

	if sessionMode == types.SessionModeInference && message == "" {
		return nil, fmt.Errorf("inference sessions require a message")
	}

	filePaths := []string{}
	files, okFiles := req.MultipartForm.File["files"]
	inputPath := controller.GetSessionInputsFolder(sessionID)

	metadata := map[string]string{}

	if okFiles {
		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				return nil, fmt.Errorf("unable to open file")
			}
			defer file.Close()

			filePath := filepath.Join(inputPath, fileHeader.Filename)

			log.Printf("uploading file %s", filePath)
			imageItem, err := apiServer.Controller.FilestoreUpload(apiServer.getRequestContext(req), filePath, file)
			if err != nil {
				return nil, fmt.Errorf("unable to upload file: %s", err.Error())
			}
			log.Printf("success uploading file: %s", imageItem.Path)
			filePaths = append(filePaths, imageItem.Path)

			// let's see if there is a single form field named after the filename
			// this is for labelling images for fine tuning
			labelValues, ok := req.MultipartForm.Value[fileHeader.Filename]

			if ok && len(labelValues) > 0 {
				filenameParts := strings.Split(fileHeader.Filename, ".")
				filenameParts[len(filenameParts)-1] = "txt"
				labelFilename := strings.Join(filenameParts, ".")
				labelFilepath := filepath.Join(inputPath, labelFilename)
				label := labelValues[0]

				metadata[fileHeader.Filename] = label

				labelItem, err := apiServer.Controller.FilestoreUpload(apiServer.getRequestContext(req), labelFilepath, strings.NewReader(label))
				if err != nil {
					return nil, fmt.Errorf("unable to create label: %s", err.Error())
				}
				log.Printf("success uploading file: %s", fileHeader.Filename)
				filePaths = append(filePaths, labelItem.Path)
			}
		}
		log.Printf("success uploading files")
	}

	if sessionMode == types.SessionModeFinetune && len(filePaths) == 0 {
		return nil, fmt.Errorf("finetune sessions require some files")
	}

	return &types.Interaction{
		ID:       system.GenerateUUID(),
		Created:  time.Now(),
		Creator:  types.CreatorTypeUser,
		Message:  message,
		Files:    filePaths,
		State:    types.InteractionStateComplete,
		Finished: true,
		Metadata: metadata,
	}, nil
}

func (apiServer *HelixAPIServer) createSession(res http.ResponseWriter, req *http.Request) (*types.Session, error) {
	reqContext := apiServer.getRequestContext(req)

	// now upload any files that were included
	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		return nil, err
	}

	sessionMode, err := types.ValidateSessionMode(req.FormValue("mode"), false)
	if err != nil {
		return nil, err
	}

	sessionType, err := types.ValidateSessionType(req.FormValue("type"), false)
	if err != nil {
		return nil, err
	}

	modelName, err := model.GetModelNameForSession(sessionType)
	if err != nil {
		return nil, err
	}

	sessionID := system.GenerateUUID()

	// the user interaction is the request from the user
	userInteraction, err := apiServer.getUserInteractionFromForm(req, sessionID, sessionMode)
	if err != nil {
		return nil, err
	}
	if userInteraction == nil {
		return nil, fmt.Errorf("no interaction found")
	}

	sessionData, err := apiServer.Controller.CreateSession(req.Context(), types.CreateSessionRequest{
		SessionID:       sessionID,
		SessionMode:     sessionMode,
		SessionType:     sessionType,
		ModelName:       modelName,
		Owner:           reqContext.Owner,
		OwnerType:       reqContext.OwnerType,
		UserInteraction: *userInteraction,
	})

	return sessionData, nil
}

func (apiServer *HelixAPIServer) updateSession(res http.ResponseWriter, req *http.Request) (*types.Session, error) {
	reqContext := apiServer.getRequestContext(req)

	// now upload any files that were included
	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		return nil, err
	}

	vars := mux.Vars(req)
	sessionID := vars["id"]
	if sessionID == "" {
		return nil, fmt.Errorf("cannot update session without id")
	}

	session, err := apiServer.Store.GetSession(req.Context(), sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("no session found with id %v", sessionID)
	}

	if session.Owner != reqContext.Owner || session.OwnerType != reqContext.OwnerType {
		return nil, fmt.Errorf("access denied")
	}

	userInteraction, err := apiServer.getUserInteractionFromForm(req, sessionID, session.Mode)
	if err != nil {
		return nil, err
	}
	if userInteraction == nil {
		return nil, fmt.Errorf("no interaction found")
	}

	sessionData, err := apiServer.Controller.UpdateSession(req.Context(), types.UpdateSessionRequest{
		SessionID:       sessionID,
		UserInteraction: *userInteraction,
	})

	return sessionData, nil
}

func (apiServer *HelixAPIServer) isAdmin(req *http.Request) bool {
	user := getRequestUser(req)
	adminUserIDs := strings.Split(os.Getenv("ADMIN_USER_IDS"), ",")
	for _, a := range adminUserIDs {
		// development mode everyone is an admin
		if a == "*" {
			return true
		}
		if a == user {
			return true
		}
	}
	return false
}

func (apiServer *HelixAPIServer) requireAdmin(req *http.Request) error {
	isAdmin := apiServer.isAdmin(req)
	if !isAdmin {
		return fmt.Errorf("access denied")
	} else {
		return nil
	}
}

// admin is required by the auth middleware
func (apiServer *HelixAPIServer) dashboard(res http.ResponseWriter, req *http.Request) (*types.DashboardData, error) {
	return apiServer.Controller.GetDashboardData(req.Context())
}

func (apiServer *HelixAPIServer) deleteSession(res http.ResponseWriter, req *http.Request) (*types.Session, error) {
	vars := mux.Vars(req)
	id := vars["id"]
	reqContext := apiServer.getRequestContext(req)

	session, err := apiServer.Store.GetSession(reqContext.Ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("no session found with id %v", id)
	}
	log.Printf("session %+v %+v", session, reqContext)
	if session.OwnerType != reqContext.OwnerType || session.Owner != reqContext.Owner {
		return nil, fmt.Errorf("access denied")
	}
	return apiServer.Store.DeleteSession(reqContext.Ctx, id)
}

func (apiServer *HelixAPIServer) getNextRunnerSession(res http.ResponseWriter, req *http.Request) (*types.Session, error) {
	vars := mux.Vars(req)
	runnerID := vars["runnerid"]
	if runnerID == "" {
		return nil, fmt.Errorf("cannot get next session without runner id")
	}

	sessionMode, err := types.ValidateSessionMode(req.URL.Query().Get("mode"), true)
	if err != nil {
		return nil, err
	}

	sessionType, err := types.ValidateSessionType(req.URL.Query().Get("type"), true)
	if err != nil {
		return nil, err
	}

	modelName, err := types.ValidateModelName(req.URL.Query().Get("model_name"), true)
	if err != nil {
		return nil, err
	}

	loraDir := req.URL.Query().Get("lora_dir")

	memory := uint64(0)
	memoryString := req.URL.Query().Get("memory")
	if memoryString != "" {
		memory, err = strconv.ParseUint(memoryString, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	// there are multiple entries for this param all of the format:
	// model_name:mode
	reject := []types.SessionFilterModel{}
	rejectPairs, ok := req.URL.Query()["reject"]

	if ok && len(rejectPairs) > 0 {
		for _, rejectPair := range rejectPairs {
			triple := strings.Split(rejectPair, ":")
			if len(triple) != 3 {
				return nil, fmt.Errorf("invalid reject pair: %s", rejectPair)
			}
			rejectModelName, err := types.ValidateModelName(triple[0], false)
			if err != nil {
				return nil, err
			}
			rejectModelMode, err := types.ValidateSessionMode(triple[1], false)
			if err != nil {
				return nil, err
			}
			rejectFinetuneFile := triple[2]
			reject = append(reject, types.SessionFilterModel{
				ModelName:    rejectModelName,
				Mode:         rejectModelMode,
				FinetuneFile: rejectFinetuneFile,
			})
		}
	}

	older := req.URL.Query().Get("older")

	var olderDuration time.Duration
	if older != "" {
		olderDuration, err = time.ParseDuration(older)
		if err != nil {
			return nil, err
		}
	}

	filter := types.SessionFilter{
		Mode:      sessionMode,
		Type:      sessionType,
		ModelName: modelName,
		Memory:    memory,
		Reject:    reject,
		LoraDir:   loraDir,
		Older:     types.Duration(olderDuration),
	}

	// alow the worker to filter what tasks it wants
	// if any of these values are defined then we will only consider those in the response
	nextSession, err := apiServer.Controller.ShiftSessionQueue(req.Context(), filter, runnerID)
	if err != nil {
		return nil, err
	}

	// IMPORTANT: we need to throw an error here (i.e. non 200 http code) because
	// that is how the workers will know to wait before asking again
	if nextSession == nil {
		return nil, fmt.Errorf("no task found")
	}

	return nextSession, nil
}

func (apiServer *HelixAPIServer) handleRunnerResponse(res http.ResponseWriter, req *http.Request) (*types.RunnerTaskResponse, error) {
	taskResponse := &types.RunnerTaskResponse{}
	err := json.NewDecoder(req.Body).Decode(taskResponse)
	if err != nil {
		return nil, err
	}

	taskResponse, err = apiServer.Controller.HandleRunnerResponse(req.Context(), taskResponse)
	if err != nil {
		return nil, err
	}
	return taskResponse, nil
}

func (apiServer *HelixAPIServer) handleRunnerMetrics(res http.ResponseWriter, req *http.Request) (*types.RunnerState, error) {
	runnerState := &types.RunnerState{}
	err := json.NewDecoder(req.Body).Decode(runnerState)
	if err != nil {
		return nil, err
	}

	runnerState, err = apiServer.Controller.AddRunnerMetrics(req.Context(), runnerState)
	if err != nil {
		return nil, err
	}
	return runnerState, nil
}

func (apiServer *HelixAPIServer) createAPIKey(res http.ResponseWriter, req *http.Request) (string, error) {
	name := req.URL.Query().Get("name")
	apiKey, err := apiServer.Controller.CreateAPIKey(apiServer.getRequestContext(req), name)
	if err != nil {
		return "", err
	}
	return apiKey, nil
}

func (apiServer *HelixAPIServer) getAPIKeys(res http.ResponseWriter, req *http.Request) ([]*types.ApiKey, error) {
	apiKeys, err := apiServer.Controller.GetAPIKeys(apiServer.getRequestContext(req))
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (apiServer *HelixAPIServer) deleteAPIKey(res http.ResponseWriter, req *http.Request) (string, error) {
	apiKey := req.URL.Query().Get("key")
	err := apiServer.Controller.DeleteAPIKey(apiServer.getRequestContext(req), apiKey)
	if err != nil {
		return "", err
	}
	return "", nil
}

func (apiServer *HelixAPIServer) checkAPIKey(res http.ResponseWriter, req *http.Request) (*types.ApiKey, error) {
	apiKey := req.URL.Query().Get("key")
	key, err := apiServer.Controller.CheckAPIKey(apiServer.getRequestContext(req).Ctx, apiKey)
	if err != nil {
		return nil, err
	}
	return key, nil
}
